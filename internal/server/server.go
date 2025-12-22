package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/aouiniamine/aoui-drive/internal/cache"
	"github.com/aouiniamine/aoui-drive/internal/config"
	"github.com/aouiniamine/aoui-drive/internal/database"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Server struct {
	echo   *echo.Echo
	config *config.Config
	db     *database.Database
	cache  *cache.Redis
}

func New(cfg *config.Config, db *database.Database, cache *cache.Redis) *Server {
	e := echo.New()
	e.HideBanner = true

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(middleware.CORS())

	return &Server{
		echo:   e,
		config: cfg,
		db:     db,
		cache:  cache,
	}
}

func (s *Server) Echo() *echo.Echo {
	return s.echo
}

func (s *Server) DB() *database.Database {
	return s.db
}

func (s *Server) Cache() *cache.Redis {
	return s.cache
}

func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%s", s.config.Server.Host, s.config.Server.Port)
	return s.echo.Start(addr)
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.echo.Shutdown(ctx)
}

func (s *Server) RegisterRoutes(register func(e *echo.Echo)) {
	register(s.echo)
}

func (s *Server) HealthCheck() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "healthy",
		})
	}
}
