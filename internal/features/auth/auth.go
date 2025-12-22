package auth

import (
	"github.com/aouiniamine/aoui-drive/internal/database"
	"github.com/aouiniamine/aoui-drive/internal/features/auth/controller"
	"github.com/aouiniamine/aoui-drive/internal/features/auth/repository"
	"github.com/aouiniamine/aoui-drive/internal/features/auth/service"
	"github.com/aouiniamine/aoui-drive/internal/middleware"
	"github.com/labstack/echo/v4"
)

type Feature struct {
	Controller *controller.AuthController
	Service    service.AuthService
}

func New(db *database.Database, jwtSecret string) *Feature {
	repo := repository.New(db.Queries)
	svc := service.New(repo, jwtSecret)
	ctrl := controller.New(svc)

	return &Feature{
		Controller: ctrl,
		Service:    svc,
	}
}

func (f *Feature) RegisterRoutes(e *echo.Echo) {
	authMiddleware := middleware.Auth(f.Service)
	adminMiddleware := middleware.RequireAdmin(f.Service)
	f.Controller.RegisterRoutes(e, authMiddleware, adminMiddleware)
}
