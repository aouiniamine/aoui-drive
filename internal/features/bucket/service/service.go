package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/aouiniamine/aoui-drive/internal/database/sqlc"
	"github.com/aouiniamine/aoui-drive/internal/features/bucket/dto"
	"github.com/aouiniamine/aoui-drive/internal/features/bucket/repository"
	"github.com/google/uuid"
)

var bucketNameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$`)

type BucketService interface {
	Create(ctx context.Context, clientID string, req dto.CreateBucketRequest) (*dto.BucketResponse, error)
	Get(ctx context.Context, clientID, name string) (*dto.BucketResponse, error)
	List(ctx context.Context, clientID string) (*dto.BucketListResponse, error)
	Delete(ctx context.Context, clientID, name string) error
}

type bucketService struct {
	repo        repository.BucketRepository
	storagePath string
}

func New(repo repository.BucketRepository, storagePath string) BucketService {
	return &bucketService{
		repo:        repo,
		storagePath: storagePath,
	}
}

func (s *bucketService) Create(ctx context.Context, clientID string, req dto.CreateBucketRequest) (*dto.BucketResponse, error) {
	if !isValidBucketName(req.Name) {
		return nil, fmt.Errorf("invalid bucket name: must be 3-63 characters, lowercase letters, numbers, hyphens, and periods")
	}

	bucketID := uuid.New().String()

	var isPublic int64
	if req.Public {
		isPublic = 1
	}

	bucket, err := s.repo.Create(ctx, sqlc.CreateBucketParams{
		ID:       bucketID,
		Name:     req.Name,
		ClientID: clientID,
		IsPublic: isPublic,
	})
	if err != nil {
		return nil, err
	}

	bucketPath := filepath.Join(s.storagePath, bucketID)
	if err := os.MkdirAll(bucketPath, 0755); err != nil {
		s.repo.Delete(ctx, bucketID)
		return nil, fmt.Errorf("failed to create bucket storage: %w", err)
	}

	// Create symlink for public bucket
	if req.Public {
		if err := s.createPublicSymlink(bucketID); err != nil {
			os.RemoveAll(bucketPath)
			s.repo.Delete(ctx, bucketID)
			return nil, fmt.Errorf("failed to create public symlink: %w", err)
		}
	}

	return &dto.BucketResponse{
		ID:        bucket.ID,
		Name:      bucket.Name,
		Public:    bucket.IsPublic == 1,
		CreatedAt: bucket.CreatedAt.Time,
	}, nil
}

func (s *bucketService) Get(ctx context.Context, clientID, name string) (*dto.BucketResponse, error) {
	bucket, err := s.repo.GetByNameAndClientID(ctx, name, clientID)
	if err != nil {
		return nil, err
	}

	return &dto.BucketResponse{
		ID:        bucket.ID,
		Name:      bucket.Name,
		Public:    bucket.IsPublic == 1,
		CreatedAt: bucket.CreatedAt.Time,
	}, nil
}

func (s *bucketService) List(ctx context.Context, clientID string) (*dto.BucketListResponse, error) {
	buckets, err := s.repo.ListByClientID(ctx, clientID)
	if err != nil {
		return nil, err
	}

	response := &dto.BucketListResponse{
		Buckets: make([]dto.BucketResponse, len(buckets)),
	}

	for i, b := range buckets {
		response.Buckets[i] = dto.BucketResponse{
			ID:        b.ID,
			Name:      b.Name,
			Public:    b.IsPublic == 1,
			CreatedAt: b.CreatedAt.Time,
		}
	}

	return response, nil
}

func (s *bucketService) Delete(ctx context.Context, clientID, name string) error {
	bucket, err := s.repo.GetByNameAndClientID(ctx, name, clientID)
	if err != nil {
		return err
	}

	bucketPath := filepath.Join(s.storagePath, bucket.ID)

	if err := s.repo.Delete(ctx, bucket.ID); err != nil {
		return err
	}

	// Remove public symlink if bucket was public
	if bucket.IsPublic == 1 {
		s.removePublicSymlink(bucket.ID)
	}

	os.RemoveAll(bucketPath)

	return nil
}

func (s *bucketService) createPublicSymlink(bucketID string) error {
	publicDir := filepath.Join(s.storagePath, "public")
	if err := os.MkdirAll(publicDir, 0755); err != nil {
		return err
	}

	// Use relative path from public folder to bucket folder
	targetPath := filepath.Join("..", bucketID)
	symlinkPath := filepath.Join(publicDir, bucketID)

	return os.Symlink(targetPath, symlinkPath)
}

func (s *bucketService) removePublicSymlink(bucketID string) {
	symlinkPath := filepath.Join(s.storagePath, "public", bucketID)
	os.Remove(symlinkPath)
}

func isValidBucketName(name string) bool {
	if len(name) < 3 || len(name) > 63 {
		return false
	}
	return bucketNameRegex.MatchString(name)
}
