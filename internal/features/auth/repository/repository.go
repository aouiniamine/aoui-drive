package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/aouiniamine/aoui-drive/internal/database/sqlc"
)

var (
	ErrClientNotFound = errors.New("client not found")
	ErrClientExists   = errors.New("client already exists")
)

type ClientRepository interface {
	GetByID(ctx context.Context, id string) (*sqlc.Client, error)
	GetByAccessKey(ctx context.Context, accessKey string) (*sqlc.Client, error)
	List(ctx context.Context) ([]sqlc.ListClientsRow, error)
	Create(ctx context.Context, params sqlc.CreateClientParams) (*sqlc.Client, error)
	Update(ctx context.Context, params sqlc.UpdateClientParams) (*sqlc.Client, error)
	UpdateSecret(ctx context.Context, id, secretKey string) error
	Delete(ctx context.Context, id string) error
	ExistsByAccessKey(ctx context.Context, accessKey string) (bool, error)
}

type clientRepository struct {
	queries *sqlc.Queries
}

func New(queries *sqlc.Queries) ClientRepository {
	return &clientRepository{queries: queries}
}

func (r *clientRepository) GetByID(ctx context.Context, id string) (*sqlc.Client, error) {
	client, err := r.queries.GetClientByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrClientNotFound
		}
		return nil, err
	}
	return &client, nil
}

func (r *clientRepository) GetByAccessKey(ctx context.Context, accessKey string) (*sqlc.Client, error) {
	client, err := r.queries.GetClientByAccessKey(ctx, accessKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrClientNotFound
		}
		return nil, err
	}
	return &client, nil
}

func (r *clientRepository) List(ctx context.Context) ([]sqlc.ListClientsRow, error) {
	return r.queries.ListClients(ctx)
}

func (r *clientRepository) Create(ctx context.Context, params sqlc.CreateClientParams) (*sqlc.Client, error) {
	exists, err := r.ExistsByAccessKey(ctx, params.AccessKey)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrClientExists
	}

	client, err := r.queries.CreateClient(ctx, params)
	if err != nil {
		return nil, err
	}
	return &client, nil
}

func (r *clientRepository) Update(ctx context.Context, params sqlc.UpdateClientParams) (*sqlc.Client, error) {
	client, err := r.queries.UpdateClient(ctx, params)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrClientNotFound
		}
		return nil, err
	}
	return &client, nil
}

func (r *clientRepository) UpdateSecret(ctx context.Context, id, secretKey string) error {
	rowsAffected, err := r.queries.UpdateClientSecret(ctx, sqlc.UpdateClientSecretParams{
		SecretKey: secretKey,
		ID:        id,
	})
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrClientNotFound
	}
	return nil
}

func (r *clientRepository) Delete(ctx context.Context, id string) error {
	return r.queries.DeleteClient(ctx, id)
}

func (r *clientRepository) ExistsByAccessKey(ctx context.Context, accessKey string) (bool, error) {
	result, err := r.queries.ClientExistsByAccessKey(ctx, accessKey)
	if err != nil {
		return false, err
	}
	return result > 0, nil
}
