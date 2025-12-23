package controller

import (
	"errors"

	bucketrepo "github.com/aouiniamine/aoui-drive/internal/features/bucket/repository"
	"github.com/aouiniamine/aoui-drive/internal/features/webhook/dto"
	"github.com/aouiniamine/aoui-drive/internal/features/webhook/repository"
	"github.com/aouiniamine/aoui-drive/internal/features/webhook/service"
	"github.com/aouiniamine/aoui-drive/internal/middleware"
	"github.com/aouiniamine/aoui-drive/pkg/response"
	"github.com/labstack/echo/v4"
)

type WebhookController struct {
	service service.WebhookService
}

func New(svc service.WebhookService) *WebhookController {
	return &WebhookController{service: svc}
}

func (c *WebhookController) RegisterRoutes(g *echo.Group) {
	// Webhook URL routes
	g.POST("", c.CreateWebhookURL)
	g.GET("", c.ListWebhookURLs)
	g.GET("/:webhookId", c.GetWebhookURL)
	g.PUT("/:webhookId", c.UpdateWebhookURL)
	g.DELETE("/:webhookId", c.DeleteWebhookURL)

	// Header routes (nested under webhook)
	g.POST("/:webhookId/headers", c.CreateHeader)
	g.PUT("/:webhookId/headers/:headerId", c.UpdateHeader)
	g.DELETE("/:webhookId/headers/:headerId", c.DeleteHeader)
}

// CreateWebhookURL godoc
// @Summary Create a webhook URL
// @Description Create a new webhook URL for a bucket
// @Tags webhooks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param bucketId path string true "Bucket ID"
// @Param request body dto.CreateWebhookURLRequest true "Webhook details"
// @Success 201 {object} response.Response{data=dto.WebhookURLResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /buckets/{bucketId}/webhooks [post]
func (c *WebhookController) CreateWebhookURL(ctx echo.Context) error {
	clientID := middleware.GetClientID(ctx)
	bucketID := ctx.Param("bucketId")

	var req dto.CreateWebhookURLRequest
	if err := ctx.Bind(&req); err != nil {
		return response.BadRequest(ctx, "invalid request body")
	}

	if req.URL == "" {
		return response.BadRequest(ctx, "url is required")
	}

	if req.EventType == "" {
		return response.BadRequest(ctx, "event_type is required")
	}

	if req.EventType != dto.EventResourceNew && req.EventType != dto.EventResourceDeleted {
		return response.BadRequest(ctx, "event_type must be 'resource.new' or 'resource.deleted'")
	}

	webhook, err := c.service.CreateURL(ctx.Request().Context(), clientID, bucketID, req)
	if err != nil {
		if errors.Is(err, bucketrepo.ErrBucketNotFound) {
			return response.NotFound(ctx, "bucket not found")
		}
		if errors.Is(err, repository.ErrWebhookURLExists) {
			return response.BadRequest(ctx, "webhook URL already exists for this event type")
		}
		if errors.Is(err, service.ErrInvalidURL) {
			return response.BadRequest(ctx, "invalid webhook URL")
		}
		return response.InternalError(ctx, err.Error())
	}

	return response.Created(ctx, webhook)
}

// ListWebhookURLs godoc
// @Summary List webhook URLs
// @Description List all webhook URLs for a bucket
// @Tags webhooks
// @Produce json
// @Security BearerAuth
// @Param bucketId path string true "Bucket ID"
// @Success 200 {object} response.Response{data=dto.WebhookURLListResponse}
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /buckets/{bucketId}/webhooks [get]
func (c *WebhookController) ListWebhookURLs(ctx echo.Context) error {
	clientID := middleware.GetClientID(ctx)
	bucketID := ctx.Param("bucketId")

	webhooks, err := c.service.ListURLs(ctx.Request().Context(), clientID, bucketID)
	if err != nil {
		if errors.Is(err, bucketrepo.ErrBucketNotFound) {
			return response.NotFound(ctx, "bucket not found")
		}
		return response.InternalError(ctx, err.Error())
	}

	return response.Success(ctx, webhooks)
}

// GetWebhookURL godoc
// @Summary Get webhook URL details
// @Description Get details of a specific webhook URL
// @Tags webhooks
// @Produce json
// @Security BearerAuth
// @Param bucketId path string true "Bucket ID"
// @Param webhookId path string true "Webhook ID"
// @Success 200 {object} response.Response{data=dto.WebhookURLResponse}
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /buckets/{bucketId}/webhooks/{webhookId} [get]
func (c *WebhookController) GetWebhookURL(ctx echo.Context) error {
	clientID := middleware.GetClientID(ctx)
	bucketID := ctx.Param("bucketId")
	webhookID := ctx.Param("webhookId")

	webhook, err := c.service.GetURL(ctx.Request().Context(), clientID, bucketID, webhookID)
	if err != nil {
		if errors.Is(err, bucketrepo.ErrBucketNotFound) {
			return response.NotFound(ctx, "bucket not found")
		}
		if errors.Is(err, repository.ErrWebhookURLNotFound) {
			return response.NotFound(ctx, "webhook not found")
		}
		return response.InternalError(ctx, err.Error())
	}

	return response.Success(ctx, webhook)
}

// UpdateWebhookURL godoc
// @Summary Update webhook URL
// @Description Update a webhook URL
// @Tags webhooks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param bucketId path string true "Bucket ID"
// @Param webhookId path string true "Webhook ID"
// @Param request body dto.UpdateWebhookURLRequest true "Webhook details"
// @Success 200 {object} response.Response{data=dto.WebhookURLResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /buckets/{bucketId}/webhooks/{webhookId} [put]
func (c *WebhookController) UpdateWebhookURL(ctx echo.Context) error {
	clientID := middleware.GetClientID(ctx)
	bucketID := ctx.Param("bucketId")
	webhookID := ctx.Param("webhookId")

	var req dto.UpdateWebhookURLRequest
	if err := ctx.Bind(&req); err != nil {
		return response.BadRequest(ctx, "invalid request body")
	}

	if req.URL == "" {
		return response.BadRequest(ctx, "url is required")
	}

	if req.EventType != dto.EventResourceNew && req.EventType != dto.EventResourceDeleted {
		return response.BadRequest(ctx, "event_type must be 'resource.new' or 'resource.deleted'")
	}

	webhook, err := c.service.UpdateURL(ctx.Request().Context(), clientID, bucketID, webhookID, req)
	if err != nil {
		if errors.Is(err, bucketrepo.ErrBucketNotFound) {
			return response.NotFound(ctx, "bucket not found")
		}
		if errors.Is(err, repository.ErrWebhookURLNotFound) {
			return response.NotFound(ctx, "webhook not found")
		}
		if errors.Is(err, service.ErrInvalidURL) {
			return response.BadRequest(ctx, "invalid webhook URL")
		}
		return response.InternalError(ctx, err.Error())
	}

	return response.Success(ctx, webhook)
}

// DeleteWebhookURL godoc
// @Summary Delete webhook URL
// @Description Delete a webhook URL
// @Tags webhooks
// @Produce json
// @Security BearerAuth
// @Param bucketId path string true "Bucket ID"
// @Param webhookId path string true "Webhook ID"
// @Success 204
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /buckets/{bucketId}/webhooks/{webhookId} [delete]
func (c *WebhookController) DeleteWebhookURL(ctx echo.Context) error {
	clientID := middleware.GetClientID(ctx)
	bucketID := ctx.Param("bucketId")
	webhookID := ctx.Param("webhookId")

	if err := c.service.DeleteURL(ctx.Request().Context(), clientID, bucketID, webhookID); err != nil {
		if errors.Is(err, bucketrepo.ErrBucketNotFound) {
			return response.NotFound(ctx, "bucket not found")
		}
		if errors.Is(err, repository.ErrWebhookURLNotFound) {
			return response.NotFound(ctx, "webhook not found")
		}
		return response.InternalError(ctx, err.Error())
	}

	return response.NoContent(ctx)
}

// CreateHeader godoc
// @Summary Create webhook header
// @Description Add a custom header to a webhook URL
// @Tags webhooks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param bucketId path string true "Bucket ID"
// @Param webhookId path string true "Webhook ID"
// @Param request body dto.CreateHeaderRequest true "Header details"
// @Success 201 {object} response.Response{data=dto.HeaderResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /buckets/{bucketId}/webhooks/{webhookId}/headers [post]
func (c *WebhookController) CreateHeader(ctx echo.Context) error {
	clientID := middleware.GetClientID(ctx)
	bucketID := ctx.Param("bucketId")
	webhookID := ctx.Param("webhookId")

	var req dto.CreateHeaderRequest
	if err := ctx.Bind(&req); err != nil {
		return response.BadRequest(ctx, "invalid request body")
	}

	if req.Name == "" {
		return response.BadRequest(ctx, "name is required")
	}

	if req.Value == "" {
		return response.BadRequest(ctx, "value is required")
	}

	header, err := c.service.CreateHeader(ctx.Request().Context(), clientID, bucketID, webhookID, req)
	if err != nil {
		if errors.Is(err, bucketrepo.ErrBucketNotFound) {
			return response.NotFound(ctx, "bucket not found")
		}
		if errors.Is(err, repository.ErrWebhookURLNotFound) {
			return response.NotFound(ctx, "webhook not found")
		}
		return response.InternalError(ctx, err.Error())
	}

	return response.Created(ctx, header)
}

// UpdateHeader godoc
// @Summary Update webhook header
// @Description Update a webhook header value
// @Tags webhooks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param bucketId path string true "Bucket ID"
// @Param webhookId path string true "Webhook ID"
// @Param headerId path string true "Header ID"
// @Param request body dto.UpdateHeaderRequest true "Header details"
// @Success 200 {object} response.Response{data=dto.HeaderResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /buckets/{bucketId}/webhooks/{webhookId}/headers/{headerId} [put]
func (c *WebhookController) UpdateHeader(ctx echo.Context) error {
	clientID := middleware.GetClientID(ctx)
	bucketID := ctx.Param("bucketId")
	webhookID := ctx.Param("webhookId")
	headerID := ctx.Param("headerId")

	var req dto.UpdateHeaderRequest
	if err := ctx.Bind(&req); err != nil {
		return response.BadRequest(ctx, "invalid request body")
	}

	if req.Value == "" {
		return response.BadRequest(ctx, "value is required")
	}

	header, err := c.service.UpdateHeader(ctx.Request().Context(), clientID, bucketID, webhookID, headerID, req)
	if err != nil {
		if errors.Is(err, bucketrepo.ErrBucketNotFound) {
			return response.NotFound(ctx, "bucket not found")
		}
		if errors.Is(err, repository.ErrWebhookURLNotFound) {
			return response.NotFound(ctx, "webhook not found")
		}
		if errors.Is(err, repository.ErrWebhookHeaderNotFound) {
			return response.NotFound(ctx, "header not found")
		}
		return response.InternalError(ctx, err.Error())
	}

	return response.Success(ctx, header)
}

// DeleteHeader godoc
// @Summary Delete webhook header
// @Description Delete a webhook header
// @Tags webhooks
// @Produce json
// @Security BearerAuth
// @Param bucketId path string true "Bucket ID"
// @Param webhookId path string true "Webhook ID"
// @Param headerId path string true "Header ID"
// @Success 204
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /buckets/{bucketId}/webhooks/{webhookId}/headers/{headerId} [delete]
func (c *WebhookController) DeleteHeader(ctx echo.Context) error {
	clientID := middleware.GetClientID(ctx)
	bucketID := ctx.Param("bucketId")
	webhookID := ctx.Param("webhookId")
	headerID := ctx.Param("headerId")

	if err := c.service.DeleteHeader(ctx.Request().Context(), clientID, bucketID, webhookID, headerID); err != nil {
		if errors.Is(err, bucketrepo.ErrBucketNotFound) {
			return response.NotFound(ctx, "bucket not found")
		}
		if errors.Is(err, repository.ErrWebhookURLNotFound) {
			return response.NotFound(ctx, "webhook not found")
		}
		if errors.Is(err, repository.ErrWebhookHeaderNotFound) {
			return response.NotFound(ctx, "header not found")
		}
		return response.InternalError(ctx, err.Error())
	}

	return response.NoContent(ctx)
}

