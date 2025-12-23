package health

import (
	"github.com/aouiniamine/aoui-drive/internal/database"
	"github.com/aouiniamine/aoui-drive/internal/features/health/controller"
	"github.com/aouiniamine/aoui-drive/internal/features/health/service"
	"github.com/labstack/echo/v4"
)

type Feature struct {
	Controller *controller.HealthController
}

func New(db *database.Database) *Feature {
	svc := service.New(db)
	ctrl := controller.New(svc)

	return &Feature{
		Controller: ctrl,
	}
}

func (f *Feature) RegisterRoutes(e *echo.Echo) {
	f.Controller.RegisterRoutes(e)
}
