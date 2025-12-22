package bucket

import (
	"github.com/aouiniamine/aoui-drive/internal/database"
	"github.com/aouiniamine/aoui-drive/internal/features/bucket/controller"
	"github.com/aouiniamine/aoui-drive/internal/features/bucket/repository"
	"github.com/aouiniamine/aoui-drive/internal/features/bucket/service"
	"github.com/labstack/echo/v4"
)

type Feature struct {
	Controller *controller.BucketController
	Service    service.BucketService
	Repository repository.BucketRepository
}

func New(db *database.Database, storagePath string) *Feature {
	repo := repository.New(db.Queries)
	svc := service.New(repo, storagePath)
	ctrl := controller.New(svc)

	return &Feature{
		Controller: ctrl,
		Service:    svc,
		Repository: repo,
	}
}

func (f *Feature) RegisterRoutes(g *echo.Group) {
	f.Controller.RegisterRoutes(g)
}
