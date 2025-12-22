package controller

import (
	"net/http"

	"github.com/aouiniamine/aoui-drive/internal/features/health/dto"
	"github.com/aouiniamine/aoui-drive/internal/features/health/service"
	"github.com/aouiniamine/aoui-drive/pkg/response"
	"github.com/labstack/echo/v4"
)

type HealthController struct {
	service service.HealthService
}

func New(svc service.HealthService) *HealthController {
	return &HealthController{
		service: svc,
	}
}

func (h *HealthController) RegisterRoutes(e *echo.Echo) {
	e.GET("/health", h.Health)
	e.GET("/ready", h.Ready)
}

func (h *HealthController) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, dto.HealthResponse{Status: "ok"})
}

func (h *HealthController) Ready(c echo.Context) error {
	status, err := h.service.Check(c.Request().Context())
	if err != nil {
		return response.InternalError(c, "failed to check health")
	}

	if status.Status != "healthy" {
		return c.JSON(http.StatusServiceUnavailable, status)
	}

	return response.Success(c, status)
}
