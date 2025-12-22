package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/aouiniamine/aoui-drive/internal/database/sqlc"
	bucketrepo "github.com/aouiniamine/aoui-drive/internal/features/bucket/repository"
	"github.com/aouiniamine/aoui-drive/internal/features/resource/dto"
	"github.com/aouiniamine/aoui-drive/internal/features/resource/repository"
	"github.com/google/uuid"
)

type ResourceService interface {
	UploadStream(ctx context.Context, clientID, bucketName, contentType string, reader io.Reader) (*dto.ResourceResponse, error)
	UploadFile(ctx context.Context, clientID, bucketName string, file *multipart.FileHeader) (*dto.ResourceResponse, error)
	Download(ctx context.Context, clientID, bucketName, hash string) (io.ReadCloser, *dto.ResourceResponse, error)
	Get(ctx context.Context, clientID, bucketName, hash string) (*dto.ResourceResponse, error)
	List(ctx context.Context, clientID, bucketName string) (*dto.ResourceListResponse, error)
	Delete(ctx context.Context, clientID, bucketName, hash string) error
}

type resourceService struct {
	repo        repository.ResourceRepository
	bucketRepo  bucketrepo.BucketRepository
	storagePath string
}

func New(repo repository.ResourceRepository, bucketRepo bucketrepo.BucketRepository, storagePath string) ResourceService {
	return &resourceService{
		repo:        repo,
		bucketRepo:  bucketRepo,
		storagePath: storagePath,
	}
}

func (s *resourceService) UploadStream(ctx context.Context, clientID, bucketName, contentType string, reader io.Reader) (*dto.ResourceResponse, error) {
	bucket, err := s.bucketRepo.GetByNameAndClientID(ctx, bucketName, clientID)
	if err != nil {
		return nil, err
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

	// Check if resource already exists (deduplication)
	existing, err := s.repo.GetByBucketAndHash(ctx, bucket.ID, hash)
	if err == nil {
		// Resource already exists, return it
		return &dto.ResourceResponse{
			ID:          existing.ID,
			Hash:        existing.Hash,
			Size:        existing.Size,
			ContentType: existing.ContentType,
			CreatedAt:   existing.CreatedAt.Time,
		}, nil
	}

	// Move temp file to final location
	resourcePath := filepath.Join(s.storagePath, bucket.ID, hash)
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
	})
	if err != nil {
		os.Remove(resourcePath)
		return nil, fmt.Errorf("failed to create resource record: %w", err)
	}

	return &dto.ResourceResponse{
		ID:          resource.ID,
		Hash:        resource.Hash,
		Size:        resource.Size,
		ContentType: resource.ContentType,
		CreatedAt:   resource.CreatedAt.Time,
	}, nil
}

func (s *resourceService) UploadFile(ctx context.Context, clientID, bucketName string, file *multipart.FileHeader) (*dto.ResourceResponse, error) {
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return s.UploadStream(ctx, clientID, bucketName, contentType, src)
}

func (s *resourceService) Download(ctx context.Context, clientID, bucketName, hash string) (io.ReadCloser, *dto.ResourceResponse, error) {
	bucket, err := s.bucketRepo.GetByNameAndClientID(ctx, bucketName, clientID)
	if err != nil {
		return nil, nil, err
	}

	resource, err := s.repo.GetByBucketAndHash(ctx, bucket.ID, hash)
	if err != nil {
		return nil, nil, err
	}

	resourcePath := filepath.Join(s.storagePath, bucket.ID, hash)
	file, err := os.Open(resourcePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open resource file: %w", err)
	}

	return file, &dto.ResourceResponse{
		ID:          resource.ID,
		Hash:        resource.Hash,
		Size:        resource.Size,
		ContentType: resource.ContentType,
		CreatedAt:   resource.CreatedAt.Time,
	}, nil
}

func (s *resourceService) Get(ctx context.Context, clientID, bucketName, hash string) (*dto.ResourceResponse, error) {
	bucket, err := s.bucketRepo.GetByNameAndClientID(ctx, bucketName, clientID)
	if err != nil {
		return nil, err
	}

	resource, err := s.repo.GetByBucketAndHash(ctx, bucket.ID, hash)
	if err != nil {
		return nil, err
	}

	return &dto.ResourceResponse{
		ID:          resource.ID,
		Hash:        resource.Hash,
		Size:        resource.Size,
		ContentType: resource.ContentType,
		CreatedAt:   resource.CreatedAt.Time,
	}, nil
}

func (s *resourceService) List(ctx context.Context, clientID, bucketName string) (*dto.ResourceListResponse, error) {
	bucket, err := s.bucketRepo.GetByNameAndClientID(ctx, bucketName, clientID)
	if err != nil {
		return nil, err
	}

	resources, err := s.repo.ListByBucketID(ctx, bucket.ID)
	if err != nil {
		return nil, err
	}

	response := &dto.ResourceListResponse{
		Resources: make([]dto.ResourceResponse, len(resources)),
	}

	for i, r := range resources {
		response.Resources[i] = dto.ResourceResponse{
			ID:          r.ID,
			Hash:        r.Hash,
			Size:        r.Size,
			ContentType: r.ContentType,
			CreatedAt:   r.CreatedAt.Time,
		}
	}

	return response, nil
}

func (s *resourceService) Delete(ctx context.Context, clientID, bucketName, hash string) error {
	bucket, err := s.bucketRepo.GetByNameAndClientID(ctx, bucketName, clientID)
	if err != nil {
		return err
	}

	if err := s.repo.DeleteByBucketAndHash(ctx, bucket.ID, hash); err != nil {
		return err
	}

	// Remove file from storage
	resourcePath := filepath.Join(s.storagePath, bucket.ID, hash)
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
