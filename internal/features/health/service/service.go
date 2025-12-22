package service

import (
	"context"

	"github.com/aouiniamine/aoui-drive/internal/cache"
	"github.com/aouiniamine/aoui-drive/internal/database"
	"github.com/aouiniamine/aoui-drive/internal/features/health/dto"
)

type HealthService interface {
	Check(ctx context.Context) (*dto.ReadyResponse, error)
}

type healthService struct {
	db    *database.Database
	cache *cache.Redis
}

func New(db *database.Database, cache *cache.Redis) HealthService {
	return &healthService{
		db:    db,
		cache: cache,
	}
}

func (s *healthService) Check(ctx context.Context) (*dto.ReadyResponse, error) {
	status := &dto.ReadyResponse{
		Status:   "healthy",
		Services: make(map[string]string),
	}

	if err := s.db.DB.PingContext(ctx); err != nil {
		status.Status = "unhealthy"
		status.Services["database"] = "unhealthy"
	} else {
		status.Services["database"] = "healthy"
	}

	if err := s.cache.Client.Ping(ctx).Err(); err != nil {
		status.Status = "unhealthy"
		status.Services["cache"] = "unhealthy"
	} else {
		status.Services["cache"] = "healthy"
	}

	return status, nil
}
