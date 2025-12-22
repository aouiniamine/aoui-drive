package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/aouiniamine/aoui-drive/internal/database/sqlc"
)

var (
	ErrResourceNotFound = errors.New("resource not found")
	ErrResourceExists   = errors.New("resource already exists")
)

type ResourceRepository interface {
	GetByID(ctx context.Context, id string) (*sqlc.Resource, error)
	GetByBucketAndHash(ctx context.Context, bucketID, hash string) (*sqlc.Resource, error)
	ListByBucketID(ctx context.Context, bucketID string) ([]sqlc.Resource, error)
	Create(ctx context.Context, params sqlc.CreateResourceParams) (*sqlc.Resource, error)
	Delete(ctx context.Context, id string) error
	DeleteByBucketAndHash(ctx context.Context, bucketID, hash string) error
	ExistsByBucketAndHash(ctx context.Context, bucketID, hash string) (bool, error)
}

type resourceRepository struct {
	queries *sqlc.Queries
}

func New(queries *sqlc.Queries) ResourceRepository {
	return &resourceRepository{queries: queries}
}

func (r *resourceRepository) GetByID(ctx context.Context, id string) (*sqlc.Resource, error) {
	resource, err := r.queries.GetResourceByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrResourceNotFound
		}
		return nil, err
	}
	return &resource, nil
}

func (r *resourceRepository) GetByBucketAndHash(ctx context.Context, bucketID, hash string) (*sqlc.Resource, error) {
	resource, err := r.queries.GetResourceByBucketAndHash(ctx, sqlc.GetResourceByBucketAndHashParams{
		BucketID: bucketID,
		Hash:     hash,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrResourceNotFound
		}
		return nil, err
	}
	return &resource, nil
}

func (r *resourceRepository) ListByBucketID(ctx context.Context, bucketID string) ([]sqlc.Resource, error) {
	return r.queries.ListResourcesByBucketID(ctx, bucketID)
}

func (r *resourceRepository) Create(ctx context.Context, params sqlc.CreateResourceParams) (*sqlc.Resource, error) {
	resource, err := r.queries.CreateResource(ctx, params)
	if err != nil {
		return nil, err
	}
	return &resource, nil
}

func (r *resourceRepository) Delete(ctx context.Context, id string) error {
	rowsAffected, err := r.queries.DeleteResource(ctx, id)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrResourceNotFound
	}
	return nil
}

func (r *resourceRepository) DeleteByBucketAndHash(ctx context.Context, bucketID, hash string) error {
	rowsAffected, err := r.queries.DeleteResourceByBucketAndHash(ctx, sqlc.DeleteResourceByBucketAndHashParams{
		BucketID: bucketID,
		Hash:     hash,
	})
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrResourceNotFound
	}
	return nil
}

func (r *resourceRepository) ExistsByBucketAndHash(ctx context.Context, bucketID, hash string) (bool, error) {
	result, err := r.queries.ResourceExistsByBucketAndHash(ctx, sqlc.ResourceExistsByBucketAndHashParams{
		BucketID: bucketID,
		Hash:     hash,
	})
	if err != nil {
		return false, err
	}
	return result > 0, nil
}
