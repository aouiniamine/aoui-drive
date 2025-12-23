package webhook

import (
	"github.com/aouiniamine/aoui-drive/internal/database"
	bucketrepo "github.com/aouiniamine/aoui-drive/internal/features/bucket/repository"
	"github.com/aouiniamine/aoui-drive/internal/features/webhook/controller"
	"github.com/aouiniamine/aoui-drive/internal/features/webhook/repository"
	"github.com/aouiniamine/aoui-drive/internal/features/webhook/service"
	"github.com/labstack/echo/v4"
)

type Feature struct {
	Controller *controller.WebhookController
	Service    service.WebhookService
	Repository repository.WebhookRepository
}

func New(db *database.Database, bucketRepo bucketrepo.BucketRepository) *Feature {
	repo := repository.New(db.Queries)
	svc := service.New(repo, bucketRepo)
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
