package controller

import (
	"errors"

	"github.com/aouiniamine/aoui-drive/internal/features/auth/dto"
	"github.com/aouiniamine/aoui-drive/internal/features/auth/repository"
	"github.com/aouiniamine/aoui-drive/internal/features/auth/service"
	"github.com/aouiniamine/aoui-drive/pkg/response"
	"github.com/labstack/echo/v4"
)

type AuthController struct {
	service service.AuthService
}

func New(svc service.AuthService) *AuthController {
	return &AuthController{service: svc}
}

func (c *AuthController) RegisterRoutes(e *echo.Echo, authMiddleware, adminMiddleware echo.MiddlewareFunc) {
	e.POST("/auth/login", c.Login)

	admin := e.Group("/admin", authMiddleware, adminMiddleware)
	admin.POST("/clients", c.CreateClient)
	admin.POST("/clients/:id/regenerate-secret", c.RegenerateSecret)
}

// Login godoc
// @Summary Authenticate client
// @Description Login with access key and secret key to get JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "Login credentials"
// @Success 200 {object} response.Response{data=dto.TokenResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /auth/login [post]
func (c *AuthController) Login(ctx echo.Context) error {
	var req dto.LoginRequest
	if err := ctx.Bind(&req); err != nil {
		return response.BadRequest(ctx, "invalid request body")
	}

	if req.AccessKey == "" || req.SecretKey == "" {
		return response.BadRequest(ctx, "access_key and secret_key are required")
	}

	token, err := c.service.Login(ctx.Request().Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			return response.Unauthorized(ctx, "invalid credentials")
		}
		if errors.Is(err, service.ErrClientInactive) {
			return response.Forbidden(ctx, "client is inactive")
		}
		return response.InternalError(ctx, "authentication failed")
	}

	return response.Success(ctx, token)
}

// CreateClient godoc
// @Summary Create a new client
// @Description Create a new client with access credentials (Admin only)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.CreateClientRequest true "Client details"
// @Success 201 {object} response.Response{data=dto.ClientResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /admin/clients [post]
func (c *AuthController) CreateClient(ctx echo.Context) error {
	var req dto.CreateClientRequest
	if err := ctx.Bind(&req); err != nil {
		return response.BadRequest(ctx, "invalid request body")
	}

	if req.Name == "" {
		return response.BadRequest(ctx, "name is required")
	}

	if req.Role == "" {
		req.Role = dto.RoleUser
	}

	if req.Role != dto.RoleAdmin && req.Role != dto.RoleManager && req.Role != dto.RoleUser {
		return response.BadRequest(ctx, "role must be ADMIN, MANAGER, or USER")
	}

	client, err := c.service.CreateClient(ctx.Request().Context(), req)
	if err != nil {
		if errors.Is(err, repository.ErrClientExists) {
			return response.BadRequest(ctx, "client already exists")
		}
		return response.InternalError(ctx, "failed to create client")
	}

	return response.Created(ctx, client)
}

// RegenerateSecret godoc
// @Summary Regenerate client secret
// @Description Regenerate the secret key for a client (Admin only)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Client ID"
// @Success 200 {object} response.Response{data=dto.SecretResponse}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /admin/clients/{id}/regenerate-secret [post]
func (c *AuthController) RegenerateSecret(ctx echo.Context) error {
	id := ctx.Param("id")

	secret, err := c.service.RegenerateSecret(ctx.Request().Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrClientNotFound) {
			return response.NotFound(ctx, "client not found")
		}
		return response.InternalError(ctx, "failed to regenerate secret")
	}

	return response.Success(ctx, secret)
}
