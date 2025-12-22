package controller

import (
	"errors"

	"github.com/aouiniamine/aoui-drive/internal/features/bucket/dto"
	"github.com/aouiniamine/aoui-drive/internal/features/bucket/repository"
	"github.com/aouiniamine/aoui-drive/internal/features/bucket/service"
	"github.com/aouiniamine/aoui-drive/internal/middleware"
	"github.com/aouiniamine/aoui-drive/pkg/response"
	"github.com/labstack/echo/v4"
)

type BucketController struct {
	service service.BucketService
}

func New(svc service.BucketService) *BucketController {
	return &BucketController{service: svc}
}

func (c *BucketController) RegisterRoutes(g *echo.Group) {
	g.POST("", c.Create)
	g.GET("", c.List)
	g.GET("/:id", c.Get)
	g.DELETE("/:id", c.Delete)
}

// Create godoc
// @Summary Create a new bucket
// @Description Create a new storage bucket for the authenticated client. If public=true, a symlink is created in the public folder.
// @Tags buckets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param public query boolean false "Make bucket publicly accessible"
// @Param request body dto.CreateBucketRequest true "Bucket details"
// @Success 201 {object} response.Response{data=dto.BucketResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /buckets [post]
func (c *BucketController) Create(ctx echo.Context) error {
	clientID := middleware.GetClientID(ctx)

	var req dto.CreateBucketRequest
	if err := ctx.Bind(&req); err != nil {
		return response.BadRequest(ctx, "invalid request body")
	}

	if req.Name == "" {
		return response.BadRequest(ctx, "name is required")
	}

	// Check public query param
	if ctx.QueryParam("public") == "true" {
		req.Public = true
	}

	bucket, err := c.service.Create(ctx.Request().Context(), clientID, req)
	if err != nil {
		if errors.Is(err, repository.ErrBucketExists) {
			return response.BadRequest(ctx, "bucket already exists")
		}
		return response.InternalError(ctx, err.Error())
	}

	return response.Created(ctx, bucket)
}

// Get godoc
// @Summary Get bucket details
// @Description Get details of a specific bucket by ID
// @Tags buckets
// @Produce json
// @Security BearerAuth
// @Param id path string true "Bucket ID"
// @Success 200 {object} response.Response{data=dto.BucketResponse}
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /buckets/{id} [get]
func (c *BucketController) Get(ctx echo.Context) error {
	clientID := middleware.GetClientID(ctx)
	bucketID := ctx.Param("id")

	bucket, err := c.service.Get(ctx.Request().Context(), clientID, bucketID)
	if err != nil {
		if errors.Is(err, repository.ErrBucketNotFound) {
			return response.NotFound(ctx, "bucket not found")
		}
		return response.InternalError(ctx, "failed to get bucket")
	}

	return response.Success(ctx, bucket)
}

// List godoc
// @Summary List all buckets
// @Description List all buckets owned by the authenticated client
// @Tags buckets
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=dto.BucketListResponse}
// @Failure 401 {object} response.Response
// @Router /buckets [get]
func (c *BucketController) List(ctx echo.Context) error {
	clientID := middleware.GetClientID(ctx)

	buckets, err := c.service.List(ctx.Request().Context(), clientID)
	if err != nil {
		return response.InternalError(ctx, "failed to list buckets")
	}

	return response.Success(ctx, buckets)
}

// Delete godoc
// @Summary Delete a bucket
// @Description Delete a bucket by ID (bucket must be empty)
// @Tags buckets
// @Produce json
// @Security BearerAuth
// @Param id path string true "Bucket ID"
// @Success 204
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /buckets/{id} [delete]
func (c *BucketController) Delete(ctx echo.Context) error {
	clientID := middleware.GetClientID(ctx)
	bucketID := ctx.Param("id")

	if err := c.service.Delete(ctx.Request().Context(), clientID, bucketID); err != nil {
		if errors.Is(err, repository.ErrBucketNotFound) {
			return response.NotFound(ctx, "bucket not found")
		}
		return response.InternalError(ctx, "failed to delete bucket")
	}

	return response.NoContent(ctx)
}
