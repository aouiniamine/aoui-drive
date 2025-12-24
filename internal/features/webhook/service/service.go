package service

import (
	"context"
	"encoding/json"
	"net/url"
	"time"

	"github.com/aouiniamine/aoui-drive/internal/database/sqlc"
	bucketrepo "github.com/aouiniamine/aoui-drive/internal/features/bucket/repository"
	"github.com/aouiniamine/aoui-drive/internal/features/webhook/dto"
	"github.com/aouiniamine/aoui-drive/internal/features/webhook/repository"
	"github.com/google/uuid"
)

type WebhookService interface {
	// Webhook URL management
	CreateURL(ctx context.Context, clientID, bucketID string, req dto.CreateWebhookURLRequest) (*dto.WebhookURLResponse, error)
	GetURL(ctx context.Context, clientID, bucketID, webhookID string) (*dto.WebhookURLResponse, error)
	ListURLs(ctx context.Context, clientID, bucketID string) (*dto.WebhookURLListResponse, error)
	UpdateURL(ctx context.Context, clientID, bucketID, webhookID string, req dto.UpdateWebhookURLRequest) (*dto.WebhookURLResponse, error)
	DeleteURL(ctx context.Context, clientID, bucketID, webhookID string) error

	// Header management
	CreateHeader(ctx context.Context, clientID, bucketID, webhookID string, req dto.CreateHeaderRequest) (*dto.HeaderResponse, error)
	UpdateHeader(ctx context.Context, clientID, bucketID, webhookID, headerID string, req dto.UpdateHeaderRequest) (*dto.HeaderResponse, error)
	DeleteHeader(ctx context.Context, clientID, bucketID, webhookID, headerID string) error

	// Event dispatching (called from resource service)
	TriggerEvent(ctx context.Context, eventType string, bucket *sqlc.Bucket, resource *sqlc.Resource, resourceURL string, extraHeaders map[string]string) error
}

type webhookService struct {
	repo       repository.WebhookRepository
	bucketRepo bucketrepo.BucketRepository
	sender     *WebhookSender
}

// Ensure webhookService implements WebhookService
var _ WebhookService = (*webhookService)(nil)

func New(repo repository.WebhookRepository, bucketRepo bucketrepo.BucketRepository) WebhookService {
	return &webhookService{
		repo:       repo,
		bucketRepo: bucketRepo,
		sender:     NewWebhookSender(repo),
	}
}

// Validation helper
func isValidURL(urlStr string) bool {
	u, err := url.Parse(urlStr)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func isValidEventType(eventType string) bool {
	return eventType == dto.EventResourceNew || eventType == dto.EventResourceDeleted
}

// verifyBucketOwnership checks if the bucket exists and belongs to the client
func (s *webhookService) verifyBucketOwnership(ctx context.Context, clientID, bucketID string) (*sqlc.Bucket, error) {
	bucket, err := s.bucketRepo.GetByID(ctx, bucketID)
	if err != nil {
		return nil, err
	}
	if bucket.ClientID != clientID {
		return nil, bucketrepo.ErrBucketNotFound
	}
	return bucket, nil
}

// verifyWebhookOwnership checks if the webhook exists and belongs to the bucket
func (s *webhookService) verifyWebhookOwnership(ctx context.Context, bucketID, webhookID string) (*sqlc.WebhookUrl, error) {
	webhook, err := s.repo.GetURLByID(ctx, webhookID)
	if err != nil {
		return nil, err
	}
	if webhook.BucketID != bucketID {
		return nil, repository.ErrWebhookURLNotFound
	}
	return webhook, nil
}

// Webhook URL management

func (s *webhookService) CreateURL(ctx context.Context, clientID, bucketID string, req dto.CreateWebhookURLRequest) (*dto.WebhookURLResponse, error) {
	if _, err := s.verifyBucketOwnership(ctx, clientID, bucketID); err != nil {
		return nil, err
	}

	if !isValidURL(req.URL) {
		return nil, ErrInvalidURL
	}

	if !isValidEventType(req.EventType) {
		return nil, ErrInvalidEventType
	}

	webhookID := uuid.New().String()
	var isActive int64
	if req.IsActive {
		isActive = 1
	}

	webhook, err := s.repo.CreateURL(ctx, sqlc.CreateWebhookURLParams{
		ID:        webhookID,
		BucketID:  bucketID,
		Url:       req.URL,
		EventType: req.EventType,
		IsActive:  isActive,
	})
	if err != nil {
		return nil, err
	}

	// Create headers if provided
	var headers []dto.HeaderResponse
	for _, h := range req.Headers {
		headerID := uuid.New().String()
		header, err := s.repo.CreateHeader(ctx, sqlc.CreateWebhookHeaderParams{
			ID:           headerID,
			WebhookUrlID: webhookID,
			HeaderName:   h.Name,
			HeaderValue:  h.Value,
		})
		if err != nil {
			continue // Skip failed headers
		}
		headers = append(headers, dto.HeaderResponse{
			ID:        header.ID,
			Name:      header.HeaderName,
			Value:     header.HeaderValue,
			CreatedAt: header.CreatedAt.Time,
		})
	}

	return &dto.WebhookURLResponse{
		ID:        webhook.ID,
		BucketID:  webhook.BucketID,
		URL:       webhook.Url,
		EventType: webhook.EventType,
		IsActive:  webhook.IsActive == 1,
		Headers:   headers,
		CreatedAt: webhook.CreatedAt.Time,
		UpdatedAt: webhook.UpdatedAt.Time,
	}, nil
}

func (s *webhookService) GetURL(ctx context.Context, clientID, bucketID, webhookID string) (*dto.WebhookURLResponse, error) {
	if _, err := s.verifyBucketOwnership(ctx, clientID, bucketID); err != nil {
		return nil, err
	}

	webhook, err := s.verifyWebhookOwnership(ctx, bucketID, webhookID)
	if err != nil {
		return nil, err
	}

	headers, err := s.repo.ListHeadersByURLID(ctx, webhookID)
	if err != nil {
		return nil, err
	}

	headerResponses := make([]dto.HeaderResponse, len(headers))
	for i, h := range headers {
		headerResponses[i] = dto.HeaderResponse{
			ID:        h.ID,
			Name:      h.HeaderName,
			Value:     h.HeaderValue,
			CreatedAt: h.CreatedAt.Time,
		}
	}

	return &dto.WebhookURLResponse{
		ID:        webhook.ID,
		BucketID:  webhook.BucketID,
		URL:       webhook.Url,
		EventType: webhook.EventType,
		IsActive:  webhook.IsActive == 1,
		Headers:   headerResponses,
		CreatedAt: webhook.CreatedAt.Time,
		UpdatedAt: webhook.UpdatedAt.Time,
	}, nil
}

func (s *webhookService) ListURLs(ctx context.Context, clientID, bucketID string) (*dto.WebhookURLListResponse, error) {
	if _, err := s.verifyBucketOwnership(ctx, clientID, bucketID); err != nil {
		return nil, err
	}

	webhooks, err := s.repo.ListURLsByBucketID(ctx, bucketID)
	if err != nil {
		return nil, err
	}

	response := &dto.WebhookURLListResponse{
		Webhooks: make([]dto.WebhookURLResponse, len(webhooks)),
	}

	for i, w := range webhooks {
		headers, _ := s.repo.ListHeadersByURLID(ctx, w.ID)
		headerResponses := make([]dto.HeaderResponse, len(headers))
		for j, h := range headers {
			headerResponses[j] = dto.HeaderResponse{
				ID:        h.ID,
				Name:      h.HeaderName,
				Value:     h.HeaderValue,
				CreatedAt: h.CreatedAt.Time,
			}
		}

		response.Webhooks[i] = dto.WebhookURLResponse{
			ID:        w.ID,
			BucketID:  w.BucketID,
			URL:       w.Url,
			EventType: w.EventType,
			IsActive:  w.IsActive == 1,
			Headers:   headerResponses,
			CreatedAt: w.CreatedAt.Time,
			UpdatedAt: w.UpdatedAt.Time,
		}
	}

	return response, nil
}

func (s *webhookService) UpdateURL(ctx context.Context, clientID, bucketID, webhookID string, req dto.UpdateWebhookURLRequest) (*dto.WebhookURLResponse, error) {
	if _, err := s.verifyBucketOwnership(ctx, clientID, bucketID); err != nil {
		return nil, err
	}

	if _, err := s.verifyWebhookOwnership(ctx, bucketID, webhookID); err != nil {
		return nil, err
	}

	if !isValidURL(req.URL) {
		return nil, ErrInvalidURL
	}

	if !isValidEventType(req.EventType) {
		return nil, ErrInvalidEventType
	}

	var isActive int64
	if req.IsActive {
		isActive = 1
	}

	webhook, err := s.repo.UpdateURL(ctx, sqlc.UpdateWebhookURLParams{
		ID:        webhookID,
		Url:       req.URL,
		EventType: req.EventType,
		IsActive:  isActive,
	})
	if err != nil {
		return nil, err
	}

	headers, _ := s.repo.ListHeadersByURLID(ctx, webhookID)
	headerResponses := make([]dto.HeaderResponse, len(headers))
	for i, h := range headers {
		headerResponses[i] = dto.HeaderResponse{
			ID:        h.ID,
			Name:      h.HeaderName,
			Value:     h.HeaderValue,
			CreatedAt: h.CreatedAt.Time,
		}
	}

	return &dto.WebhookURLResponse{
		ID:        webhook.ID,
		BucketID:  webhook.BucketID,
		URL:       webhook.Url,
		EventType: webhook.EventType,
		IsActive:  webhook.IsActive == 1,
		Headers:   headerResponses,
		CreatedAt: webhook.CreatedAt.Time,
		UpdatedAt: webhook.UpdatedAt.Time,
	}, nil
}

func (s *webhookService) DeleteURL(ctx context.Context, clientID, bucketID, webhookID string) error {
	if _, err := s.verifyBucketOwnership(ctx, clientID, bucketID); err != nil {
		return err
	}

	if _, err := s.verifyWebhookOwnership(ctx, bucketID, webhookID); err != nil {
		return err
	}

	return s.repo.DeleteURL(ctx, webhookID)
}

// Header management

func (s *webhookService) CreateHeader(ctx context.Context, clientID, bucketID, webhookID string, req dto.CreateHeaderRequest) (*dto.HeaderResponse, error) {
	if _, err := s.verifyBucketOwnership(ctx, clientID, bucketID); err != nil {
		return nil, err
	}

	if _, err := s.verifyWebhookOwnership(ctx, bucketID, webhookID); err != nil {
		return nil, err
	}

	headerID := uuid.New().String()
	header, err := s.repo.CreateHeader(ctx, sqlc.CreateWebhookHeaderParams{
		ID:           headerID,
		WebhookUrlID: webhookID,
		HeaderName:   req.Name,
		HeaderValue:  req.Value,
	})
	if err != nil {
		return nil, err
	}

	return &dto.HeaderResponse{
		ID:        header.ID,
		Name:      header.HeaderName,
		Value:     header.HeaderValue,
		CreatedAt: header.CreatedAt.Time,
	}, nil
}

func (s *webhookService) UpdateHeader(ctx context.Context, clientID, bucketID, webhookID, headerID string, req dto.UpdateHeaderRequest) (*dto.HeaderResponse, error) {
	if _, err := s.verifyBucketOwnership(ctx, clientID, bucketID); err != nil {
		return nil, err
	}

	if _, err := s.verifyWebhookOwnership(ctx, bucketID, webhookID); err != nil {
		return nil, err
	}

	// Verify header belongs to webhook
	existingHeader, err := s.repo.GetHeaderByID(ctx, headerID)
	if err != nil {
		return nil, err
	}
	if existingHeader.WebhookUrlID != webhookID {
		return nil, repository.ErrWebhookHeaderNotFound
	}

	header, err := s.repo.UpdateHeader(ctx, sqlc.UpdateWebhookHeaderParams{
		ID:          headerID,
		HeaderValue: req.Value,
	})
	if err != nil {
		return nil, err
	}

	return &dto.HeaderResponse{
		ID:        header.ID,
		Name:      header.HeaderName,
		Value:     header.HeaderValue,
		CreatedAt: header.CreatedAt.Time,
	}, nil
}

func (s *webhookService) DeleteHeader(ctx context.Context, clientID, bucketID, webhookID, headerID string) error {
	if _, err := s.verifyBucketOwnership(ctx, clientID, bucketID); err != nil {
		return err
	}

	if _, err := s.verifyWebhookOwnership(ctx, bucketID, webhookID); err != nil {
		return err
	}

	// Verify header belongs to webhook
	existingHeader, err := s.repo.GetHeaderByID(ctx, headerID)
	if err != nil {
		return err
	}
	if existingHeader.WebhookUrlID != webhookID {
		return repository.ErrWebhookHeaderNotFound
	}

	return s.repo.DeleteHeader(ctx, headerID)
}

// TriggerEvent sends webhooks directly to all active webhook URLs matching the event type
// extraHeaders are optional headers passed at request time that will be included in the webhook request
func (s *webhookService) TriggerEvent(ctx context.Context, eventType string, bucket *sqlc.Bucket, resource *sqlc.Resource, resourceURL string, extraHeaders map[string]string) error {
	webhooks, err := s.repo.ListActiveURLsByBucketAndEvent(ctx, bucket.ID, eventType)
	if err != nil {
		return err
	}

	if len(webhooks) == 0 {
		return nil // No webhooks configured
	}

	// Build payload
	payload := dto.WebhookPayload{
		Event:       eventType,
		Timestamp:   time.Now().UTC(),
		BucketID:    bucket.ID,
		BucketName:  bucket.Name,
		ResourceID:  resource.ID,
		ResourceURL: resourceURL,
		Resource: dto.ResourcePayload{
			Hash:        resource.Hash,
			Size:        resource.Size,
			ContentType: resource.ContentType,
			Extension:   resource.Extension,
		},
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Send webhook to each URL directly (fire and forget)
	for _, webhook := range webhooks {
		go func(w sqlc.WebhookUrl) {
			s.sender.SendWebhook(ctx, &w, string(payloadJSON), extraHeaders)
		}(webhook)
	}

	return nil
}

// Service errors
var (
	ErrInvalidURL       = repositoryError("invalid webhook URL")
	ErrInvalidEventType = repositoryError("invalid event type")
)

type repositoryError string

func (e repositoryError) Error() string {
	return string(e)
}
