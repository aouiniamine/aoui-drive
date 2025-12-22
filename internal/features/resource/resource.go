package resource

import (
	"github.com/aouiniamine/aoui-drive/internal/database"
	bucketrepo "github.com/aouiniamine/aoui-drive/internal/features/bucket/repository"
	"github.com/aouiniamine/aoui-drive/internal/features/resource/controller"
	"github.com/aouiniamine/aoui-drive/internal/features/resource/repository"
	"github.com/aouiniamine/aoui-drive/internal/features/resource/service"
	"github.com/labstack/echo/v4"
)

type Feature struct {
	Controller *controller.ResourceController
	Service    service.ResourceService
}

func New(db *database.Database, bucketRepo bucketrepo.BucketRepository, storagePath, publicURL string) *Feature {
	repo := repository.New(db.Queries)
	svc := service.New(repo, bucketRepo, storagePath, publicURL)
	ctrl := controller.New(svc)

	return &Feature{
		Controller: ctrl,
		Service:    svc,
	}
}

func (f *Feature) RegisterRoutes(g *echo.Group) {
	f.Controller.RegisterRoutes(g)
}
