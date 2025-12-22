package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/aouiniamine/aoui-drive/internal/database/sqlc"
)

var (
	ErrBucketNotFound = errors.New("bucket not found")
	ErrBucketExists   = errors.New("bucket already exists")
)

type BucketRepository interface {
	GetByID(ctx context.Context, id string) (*sqlc.Bucket, error)
	GetByNameAndClientID(ctx context.Context, name, clientID string) (*sqlc.Bucket, error)
	List(ctx context.Context) ([]sqlc.Bucket, error)
	ListByClientID(ctx context.Context, clientID string) ([]sqlc.Bucket, error)
	Create(ctx context.Context, params sqlc.CreateBucketParams) (*sqlc.Bucket, error)
	Delete(ctx context.Context, id string) error
	ExistsByNameAndClientID(ctx context.Context, name, clientID string) (bool, error)
}

type bucketRepository struct {
	queries *sqlc.Queries
}

func New(queries *sqlc.Queries) BucketRepository {
	return &bucketRepository{queries: queries}
}

func (r *bucketRepository) GetByID(ctx context.Context, id string) (*sqlc.Bucket, error) {
	bucket, err := r.queries.GetBucketByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrBucketNotFound
		}
		return nil, err
	}
	return &bucket, nil
}

func (r *bucketRepository) GetByNameAndClientID(ctx context.Context, name, clientID string) (*sqlc.Bucket, error) {
	bucket, err := r.queries.GetBucketByNameAndClientID(ctx, sqlc.GetBucketByNameAndClientIDParams{
		Name:     name,
		ClientID: clientID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrBucketNotFound
		}
		return nil, err
	}
	return &bucket, nil
}

func (r *bucketRepository) List(ctx context.Context) ([]sqlc.Bucket, error) {
	return r.queries.ListBuckets(ctx)
}

func (r *bucketRepository) ListByClientID(ctx context.Context, clientID string) ([]sqlc.Bucket, error) {
	return r.queries.ListBucketsByClientID(ctx, clientID)
}

func (r *bucketRepository) Create(ctx context.Context, params sqlc.CreateBucketParams) (*sqlc.Bucket, error) {
	exists, err := r.ExistsByNameAndClientID(ctx, params.Name, params.ClientID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrBucketExists
	}

	bucket, err := r.queries.CreateBucket(ctx, params)
	if err != nil {
		return nil, err
	}
	return &bucket, nil
}

func (r *bucketRepository) Delete(ctx context.Context, id string) error {
	rowsAffected, err := r.queries.DeleteBucket(ctx, id)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrBucketNotFound
	}
	return nil
}

func (r *bucketRepository) ExistsByNameAndClientID(ctx context.Context, name, clientID string) (bool, error) {
	result, err := r.queries.BucketExistsByNameAndClientID(ctx, sqlc.BucketExistsByNameAndClientIDParams{
		Name:     name,
		ClientID: clientID,
	})
	if err != nil {
		return false, err
	}
	return result > 0, nil
}
