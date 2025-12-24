package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/aouiniamine/aoui-drive/internal/database/sqlc"
	bucketrepo "github.com/aouiniamine/aoui-drive/internal/features/bucket/repository"
	"github.com/aouiniamine/aoui-drive/internal/features/resource/dto"
	"github.com/aouiniamine/aoui-drive/internal/features/resource/repository"
	webhookdto "github.com/aouiniamine/aoui-drive/internal/features/webhook/dto"
	"github.com/google/uuid"
)

// WebhookLauncher is an interface to avoid circular dependencies
type WebhookLauncher interface {
	TriggerEvent(ctx context.Context, eventType string, bucket *sqlc.Bucket, resource *sqlc.Resource, resourceURL string, extraHeaders map[string]string) error
}

type ResourceService interface {
	UploadStream(ctx context.Context, clientID, bucketID, contentType, extension string, reader io.Reader, webhookHeaders map[string]string) (*dto.ResourceResponse, error)
	UploadFile(ctx context.Context, clientID, bucketID string, file *multipart.FileHeader, webhookHeaders map[string]string) (*dto.ResourceResponse, error)
	Download(ctx context.Context, clientID, bucketID, hash string) (io.ReadCloser, *dto.ResourceResponse, error)
	Get(ctx context.Context, clientID, bucketID, hash string) (*dto.ResourceResponse, error)
	List(ctx context.Context, clientID, bucketID string) (*dto.ResourceListResponse, error)
	Delete(ctx context.Context, clientID, bucketID, hash string) error
}

type resourceService struct {
	repo            repository.ResourceRepository
	bucketRepo      bucketrepo.BucketRepository
	webhookLauncher WebhookLauncher
	storagePath     string
	publicURL       string
}

func New(repo repository.ResourceRepository, bucketRepo bucketrepo.BucketRepository, storagePath, publicURL string, webhookLauncher WebhookLauncher) ResourceService {
	return &resourceService{
		repo:            repo,
		bucketRepo:      bucketRepo,
		storagePath:     storagePath,
		publicURL:       publicURL,
		webhookLauncher: webhookLauncher,
	}
}

func (s *resourceService) UploadStream(ctx context.Context, clientID, bucketID, contentType, extension string, reader io.Reader, webhookHeaders map[string]string) (*dto.ResourceResponse, error) {
	bucket, err := s.bucketRepo.GetByID(ctx, bucketID)
	if err != nil {
		return nil, err
	}

	// Verify bucket belongs to client
	if bucket.ClientID != clientID {
		return nil, bucketrepo.ErrBucketNotFound
	}

	// Create temp file to compute hash while reading
	tempFile, err := os.CreateTemp("", "resource-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	// Compute hash while copying to temp file
	hasher := sha256.New()
	teeReader := io.TeeReader(reader, hasher)

	size, err := io.Copy(tempFile, teeReader)
	if err != nil {
		tempFile.Close()
		return nil, fmt.Errorf("failed to read content: %w", err)
	}
	tempFile.Close()

	hash := hex.EncodeToString(hasher.Sum(nil))

	// Use provided extension or fall back to content type
	ext := extension
	if ext == "" {
		var err error
		ext, err = getExtensionFromContentType(contentType)
		if err != nil {
			return nil, err
		}
	}
	if ext != "" && ext[0] != '.' {
		ext = "." + ext
	}

	// Check if resource already exists (deduplication)
	existing, err := s.repo.GetByBucketAndHash(ctx, bucket.ID, hash)
	if err == nil {
		// Resource already exists, return it
		resp := &dto.ResourceResponse{
			ID:          existing.ID,
			Hash:        existing.Hash,
			Size:        existing.Size,
			ContentType: existing.ContentType,
			Extension:   existing.Extension,
			CreatedAt:   existing.CreatedAt.Time,
		}
		if bucket.IsPublic == 1 {
			resp.PublicURL = s.buildPublicURL(bucket.ID, existing.Hash, existing.Extension)
		}
		return resp, nil
	}

	// Move temp file to final location (with extension)
	filename := buildFilename(hash, ext)
	resourcePath := filepath.Join(s.storagePath, bucket.ID, filename)
	if err := os.Rename(tempPath, resourcePath); err != nil {
		// If rename fails (cross-device), copy instead
		if err := copyFile(tempPath, resourcePath); err != nil {
			return nil, fmt.Errorf("failed to store resource: %w", err)
		}
	}

	// Create database record
	resourceID := uuid.New().String()
	resource, err := s.repo.Create(ctx, sqlc.CreateResourceParams{
		ID:          resourceID,
		BucketID:    bucket.ID,
		Hash:        hash,
		Size:        size,
		ContentType: contentType,
		Extension:   ext,
	})
	if err != nil {
		os.Remove(resourcePath)
		return nil, fmt.Errorf("failed to create resource record: %w", err)
	}

	resp := &dto.ResourceResponse{
		ID:          resource.ID,
		Hash:        resource.Hash,
		Size:        resource.Size,
		ContentType: resource.ContentType,
		Extension:   resource.Extension,
		CreatedAt:   resource.CreatedAt.Time,
	}
	if bucket.IsPublic == 1 {
		resp.PublicURL = s.buildPublicURL(bucket.ID, resource.Hash, resource.Extension)
	}

	// Trigger webhook event for new resource
	if s.webhookLauncher != nil {
		go func() {
			triggerCtx := context.Background()
			resourceURL := s.buildDownloadURL(bucket.ID, resource.Hash, resource.Extension)
			s.webhookLauncher.TriggerEvent(triggerCtx, webhookdto.EventResourceNew, bucket, resource, resourceURL, webhookHeaders)
		}()
	}

	return resp, nil
}

func (s *resourceService) UploadFile(ctx context.Context, clientID, bucketID string, file *multipart.FileHeader, webhookHeaders map[string]string) (*dto.ResourceResponse, error) {
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Extract extension from original filename
	extension := filepath.Ext(file.Filename)

	return s.UploadStream(ctx, clientID, bucketID, contentType, extension, src, webhookHeaders)
}

func (s *resourceService) Download(ctx context.Context, clientID, bucketID, hash string) (io.ReadCloser, *dto.ResourceResponse, error) {
	bucket, err := s.bucketRepo.GetByID(ctx, bucketID)
	if err != nil {
		return nil, nil, err
	}

	// Verify bucket belongs to client
	if bucket.ClientID != clientID {
		return nil, nil, bucketrepo.ErrBucketNotFound
	}

	resource, err := s.repo.GetByBucketAndHash(ctx, bucketID, hash)
	if err != nil {
		return nil, nil, err
	}

	filename := buildFilename(resource.Hash, resource.Extension)
	resourcePath := filepath.Join(s.storagePath, bucket.ID, filename)
	file, err := os.Open(resourcePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open resource file: %w", err)
	}

	resp := &dto.ResourceResponse{
		ID:          resource.ID,
		Hash:        resource.Hash,
		Size:        resource.Size,
		ContentType: resource.ContentType,
		Extension:   resource.Extension,
		CreatedAt:   resource.CreatedAt.Time,
	}
	if bucket.IsPublic == 1 {
		resp.PublicURL = s.buildPublicURL(bucket.ID, resource.Hash, resource.Extension)
	}
	return file, resp, nil
}

func (s *resourceService) Get(ctx context.Context, clientID, bucketID, hash string) (*dto.ResourceResponse, error) {
	bucket, err := s.bucketRepo.GetByID(ctx, bucketID)
	if err != nil {
		return nil, err
	}

	// Verify bucket belongs to client
	if bucket.ClientID != clientID {
		return nil, bucketrepo.ErrBucketNotFound
	}

	resource, err := s.repo.GetByBucketAndHash(ctx, bucketID, hash)
	if err != nil {
		return nil, err
	}

	resp := &dto.ResourceResponse{
		ID:          resource.ID,
		Hash:        resource.Hash,
		Size:        resource.Size,
		ContentType: resource.ContentType,
		Extension:   resource.Extension,
		CreatedAt:   resource.CreatedAt.Time,
	}
	if bucket.IsPublic == 1 {
		resp.PublicURL = s.buildPublicURL(bucket.ID, resource.Hash, resource.Extension)
	}
	return resp, nil
}

func (s *resourceService) List(ctx context.Context, clientID, bucketID string) (*dto.ResourceListResponse, error) {
	bucket, err := s.bucketRepo.GetByID(ctx, bucketID)
	if err != nil {
		return nil, err
	}

	// Verify bucket belongs to client
	if bucket.ClientID != clientID {
		return nil, bucketrepo.ErrBucketNotFound
	}

	resources, err := s.repo.ListByBucketID(ctx, bucketID)
	if err != nil {
		return nil, err
	}

	response := &dto.ResourceListResponse{
		Resources: make([]dto.ResourceResponse, len(resources)),
	}

	for i, r := range resources {
		resp := dto.ResourceResponse{
			ID:          r.ID,
			Hash:        r.Hash,
			Size:        r.Size,
			ContentType: r.ContentType,
			Extension:   r.Extension,
			CreatedAt:   r.CreatedAt.Time,
		}
		if bucket.IsPublic == 1 {
			resp.PublicURL = s.buildPublicURL(bucket.ID, r.Hash, r.Extension)
		}
		response.Resources[i] = resp
	}

	return response, nil
}

func (s *resourceService) buildPublicURL(bucketID, hash, extension string) string {
	filename := buildFilename(hash, extension)
	if s.publicURL != "" {
		return fmt.Sprintf("%s/public/%s/%s", s.publicURL, bucketID, filename)
	}
	return fmt.Sprintf("/public/%s/%s", bucketID, filename)
}

// buildDownloadURL constructs the download endpoint URL (works for both public and private buckets)
func (s *resourceService) buildDownloadURL(bucketID, hash string, extension string) string {
	if s.publicURL != "" {
		return fmt.Sprintf("%s/resources/%s/%s%s", s.publicURL, bucketID, hash, extension)
	}
	return fmt.Sprintf("/resources/%s/%s%s", bucketID, hash, extension)
}

func (s *resourceService) Delete(ctx context.Context, clientID, bucketID, hash string) error {
	bucket, err := s.bucketRepo.GetByID(ctx, bucketID)
	if err != nil {
		return err
	}

	// Verify bucket belongs to client
	if bucket.ClientID != clientID {
		return bucketrepo.ErrBucketNotFound
	}

	resource, err := s.repo.GetByBucketAndHash(ctx, bucketID, hash)
	if err != nil {
		return err
	}

	// Trigger webhook event for deleted resource before deletion
	if s.webhookLauncher != nil {
		resourceURL := s.buildDownloadURL(bucket.ID, resource.Hash, resource.Extension)
		// Create a copy of the resource for the webhook since it will be deleted
		resourceCopy := &sqlc.Resource{
			ID:          resource.ID,
			BucketID:    resource.BucketID,
			Hash:        resource.Hash,
			Size:        resource.Size,
			ContentType: resource.ContentType,
			Extension:   resource.Extension,
			CreatedAt:   resource.CreatedAt,
		}
		go func() {
			triggerCtx := context.Background()
			s.webhookLauncher.TriggerEvent(triggerCtx, webhookdto.EventResourceDeleted, bucket, resourceCopy, resourceURL, nil)
		}()
	}

	if err := s.repo.DeleteByBucketAndHash(ctx, bucketID, hash); err != nil {
		return err
	}

	// Remove file from storage
	filename := buildFilename(resource.Hash, resource.Extension)
	resourcePath := filepath.Join(s.storagePath, bucket.ID, filename)
	os.Remove(resourcePath)

	return nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func getExtensionFromContentType(contentType string) (string, error) {
	exts, err := mime.ExtensionsByType(contentType)
	if err != nil {
		return "", err
	}
	if len(exts) == 0 {
		return "", errors.New("file extension not found")
	}
	return exts[0], nil
}

func buildFilename(hash, extension string) string {
	if extension != "" {
		return hash + extension
	}
	return hash
}
