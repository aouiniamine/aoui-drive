package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/aouiniamine/aoui-drive/internal/database/sqlc"
)

var (
	ErrWebhookURLNotFound    = errors.New("webhook URL not found")
	ErrWebhookURLExists      = errors.New("webhook URL already exists for this event type")
	ErrWebhookHeaderNotFound = errors.New("webhook header not found")
	ErrWebhookEventNotFound  = errors.New("webhook event not found")
)

type WebhookRepository interface {
	// Webhook URLs
	GetURLByID(ctx context.Context, id string) (*sqlc.WebhookUrl, error)
	ListURLsByBucketID(ctx context.Context, bucketID string) ([]sqlc.WebhookUrl, error)
	ListActiveURLsByBucketAndEvent(ctx context.Context, bucketID, eventType string) ([]sqlc.WebhookUrl, error)
	CreateURL(ctx context.Context, params sqlc.CreateWebhookURLParams) (*sqlc.WebhookUrl, error)
	UpdateURL(ctx context.Context, params sqlc.UpdateWebhookURLParams) (*sqlc.WebhookUrl, error)
	DeleteURL(ctx context.Context, id string) error
	URLExists(ctx context.Context, bucketID, url, eventType string) (bool, error)

	// Webhook Headers
	GetHeaderByID(ctx context.Context, id string) (*sqlc.WebhookHeader, error)
	ListHeadersByURLID(ctx context.Context, webhookURLID string) ([]sqlc.WebhookHeader, error)
	CreateHeader(ctx context.Context, params sqlc.CreateWebhookHeaderParams) (*sqlc.WebhookHeader, error)
	UpdateHeader(ctx context.Context, params sqlc.UpdateWebhookHeaderParams) (*sqlc.WebhookHeader, error)
	DeleteHeader(ctx context.Context, id string) error
	DeleteHeadersByURLID(ctx context.Context, webhookURLID string) error

	// Webhook Events
	GetEventByID(ctx context.Context, id string) (*sqlc.WebhookEvent, error)
	ListEventsByBucketID(ctx context.Context, bucketID string, limit, offset int64) ([]sqlc.WebhookEvent, error)
	ListPendingEvents(ctx context.Context, limit int64) ([]sqlc.WebhookEvent, error)
	CreateEvent(ctx context.Context, params sqlc.CreateWebhookEventParams) (*sqlc.WebhookEvent, error)
	UpdateEventStatus(ctx context.Context, params sqlc.UpdateWebhookEventStatusParams) error
	CountEventsByBucketID(ctx context.Context, bucketID string) (int64, error)
}

type webhookRepository struct {
	queries *sqlc.Queries
}

func New(queries *sqlc.Queries) WebhookRepository {
	return &webhookRepository{queries: queries}
}

// Webhook URLs

func (r *webhookRepository) GetURLByID(ctx context.Context, id string) (*sqlc.WebhookUrl, error) {
	url, err := r.queries.GetWebhookURLByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrWebhookURLNotFound
		}
		return nil, err
	}
	return &url, nil
}

func (r *webhookRepository) ListURLsByBucketID(ctx context.Context, bucketID string) ([]sqlc.WebhookUrl, error) {
	return r.queries.ListWebhookURLsByBucketID(ctx, bucketID)
}

func (r *webhookRepository) ListActiveURLsByBucketAndEvent(ctx context.Context, bucketID, eventType string) ([]sqlc.WebhookUrl, error) {
	return r.queries.ListActiveWebhookURLsByBucketAndEvent(ctx, sqlc.ListActiveWebhookURLsByBucketAndEventParams{
		BucketID:  bucketID,
		EventType: eventType,
	})
}

func (r *webhookRepository) CreateURL(ctx context.Context, params sqlc.CreateWebhookURLParams) (*sqlc.WebhookUrl, error) {
	exists, err := r.URLExists(ctx, params.BucketID, params.Url, params.EventType)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrWebhookURLExists
	}

	url, err := r.queries.CreateWebhookURL(ctx, params)
	if err != nil {
		return nil, err
	}
	return &url, nil
}

func (r *webhookRepository) UpdateURL(ctx context.Context, params sqlc.UpdateWebhookURLParams) (*sqlc.WebhookUrl, error) {
	url, err := r.queries.UpdateWebhookURL(ctx, params)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrWebhookURLNotFound
		}
		return nil, err
	}
	return &url, nil
}

func (r *webhookRepository) DeleteURL(ctx context.Context, id string) error {
	rowsAffected, err := r.queries.DeleteWebhookURL(ctx, id)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrWebhookURLNotFound
	}
	return nil
}

func (r *webhookRepository) URLExists(ctx context.Context, bucketID, url, eventType string) (bool, error) {
	result, err := r.queries.WebhookURLExists(ctx, sqlc.WebhookURLExistsParams{
		BucketID:  bucketID,
		Url:       url,
		EventType: eventType,
	})
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

// Webhook Headers

func (r *webhookRepository) GetHeaderByID(ctx context.Context, id string) (*sqlc.WebhookHeader, error) {
	header, err := r.queries.GetWebhookHeaderByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrWebhookHeaderNotFound
		}
		return nil, err
	}
	return &header, nil
}

func (r *webhookRepository) ListHeadersByURLID(ctx context.Context, webhookURLID string) ([]sqlc.WebhookHeader, error) {
	return r.queries.ListWebhookHeadersByURLID(ctx, webhookURLID)
}

func (r *webhookRepository) CreateHeader(ctx context.Context, params sqlc.CreateWebhookHeaderParams) (*sqlc.WebhookHeader, error) {
	header, err := r.queries.CreateWebhookHeader(ctx, params)
	if err != nil {
		return nil, err
	}
	return &header, nil
}

func (r *webhookRepository) UpdateHeader(ctx context.Context, params sqlc.UpdateWebhookHeaderParams) (*sqlc.WebhookHeader, error) {
	header, err := r.queries.UpdateWebhookHeader(ctx, params)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrWebhookHeaderNotFound
		}
		return nil, err
	}
	return &header, nil
}

func (r *webhookRepository) DeleteHeader(ctx context.Context, id string) error {
	rowsAffected, err := r.queries.DeleteWebhookHeader(ctx, id)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrWebhookHeaderNotFound
	}
	return nil
}

func (r *webhookRepository) DeleteHeadersByURLID(ctx context.Context, webhookURLID string) error {
	_, err := r.queries.DeleteWebhookHeadersByURLID(ctx, webhookURLID)
	return err
}

// Webhook Events

func (r *webhookRepository) GetEventByID(ctx context.Context, id string) (*sqlc.WebhookEvent, error) {
	event, err := r.queries.GetWebhookEventByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrWebhookEventNotFound
		}
		return nil, err
	}
	return &event, nil
}

func (r *webhookRepository) ListEventsByBucketID(ctx context.Context, bucketID string, limit, offset int64) ([]sqlc.WebhookEvent, error) {
	return r.queries.ListWebhookEventsByBucketID(ctx, sqlc.ListWebhookEventsByBucketIDParams{
		BucketID: bucketID,
		Limit:    limit,
		Offset:   offset,
	})
}

func (r *webhookRepository) ListPendingEvents(ctx context.Context, limit int64) ([]sqlc.WebhookEvent, error) {
	return r.queries.ListPendingWebhookEvents(ctx, limit)
}

func (r *webhookRepository) CreateEvent(ctx context.Context, params sqlc.CreateWebhookEventParams) (*sqlc.WebhookEvent, error) {
	event, err := r.queries.CreateWebhookEvent(ctx, params)
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *webhookRepository) UpdateEventStatus(ctx context.Context, params sqlc.UpdateWebhookEventStatusParams) error {
	return r.queries.UpdateWebhookEventStatus(ctx, params)
}

func (r *webhookRepository) CountEventsByBucketID(ctx context.Context, bucketID string) (int64, error) {
	return r.queries.CountWebhookEventsByBucketID(ctx, bucketID)
}
