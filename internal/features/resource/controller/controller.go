package controller

import (
	"errors"
	"fmt"
	"net/http"

	bucketrepo "github.com/aouiniamine/aoui-drive/internal/features/bucket/repository"
	"github.com/aouiniamine/aoui-drive/internal/features/resource/repository"
	"github.com/aouiniamine/aoui-drive/internal/features/resource/service"
	"github.com/aouiniamine/aoui-drive/internal/middleware"
	"github.com/aouiniamine/aoui-drive/pkg/response"
	"github.com/labstack/echo/v4"
)

type ResourceController struct {
	service service.ResourceService
}

func New(svc service.ResourceService) *ResourceController {
	return &ResourceController{service: svc}
}

func (c *ResourceController) RegisterRoutes(g *echo.Group) {
	g.PUT("/:bucket", c.UploadStream)
	g.POST("/:bucket", c.UploadFile)
	g.GET("/:bucket/:hash", c.Download)
	g.HEAD("/:bucket/:hash", c.Head)
	g.GET("/:bucket", c.List)
	g.DELETE("/:bucket/:hash", c.Delete)
}

// UploadStream godoc
// @Summary Upload resource via stream
// @Description Upload a resource to a bucket using request body stream. The file hash (SHA-256) becomes the resource identifier for deduplication. Use X-File-Extension header to specify the file extension (e.g., ".jpg", ".log").
// @Tags resources
// @Accept */*
// @Produce json
// @Security BearerAuth
// @Param bucket path string true "Bucket ID"
// @Param X-File-Extension header string true "File extension (e.g., .jpg, .log)"
// @Param file body string true "File content" format(binary)
// @Success 200 {object} response.Response{data=dto.ResourceResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /resources/{bucket} [put]
func (c *ResourceController) UploadStream(ctx echo.Context) error {
	clientID := middleware.GetClientID(ctx)
	bucketID := ctx.Param("bucket")

	contentType := ctx.Request().Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	extension := ctx.Request().Header.Get("X-File-Extension")
	if extension == "" {
		return response.BadRequest(ctx, "X-File-Extension header is required")
	}

	resource, err := c.service.UploadStream(ctx.Request().Context(), clientID, bucketID, contentType, extension, ctx.Request().Body)
	if err != nil {
		if errors.Is(err, bucketrepo.ErrBucketNotFound) {
			return response.NotFound(ctx, "bucket not found")
		}
		return response.InternalError(ctx, err.Error())
	}

	return response.Success(ctx, resource)
}

// UploadFile godoc
// @Summary Upload resource via multipart form
// @Description Upload a resource to a bucket using multipart form file upload. The file hash (SHA-256) becomes the resource identifier for deduplication.
// @Tags resources
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param bucket path string true "Bucket ID"
// @Param file formData file true "File to upload"
// @Success 200 {object} response.Response{data=dto.ResourceResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /resources/{bucket} [post]
func (c *ResourceController) UploadFile(ctx echo.Context) error {
	clientID := middleware.GetClientID(ctx)
	bucketID := ctx.Param("bucket")

	file, err := ctx.FormFile("file")
	if err != nil {
		return response.BadRequest(ctx, "file is required")
	}

	resource, err := c.service.UploadFile(ctx.Request().Context(), clientID, bucketID, file)
	if err != nil {
		if errors.Is(err, bucketrepo.ErrBucketNotFound) {
			return response.NotFound(ctx, "bucket not found")
		}
		return response.InternalError(ctx, err.Error())
	}

	return response.Success(ctx, resource)
}

// Download godoc
// @Summary Download a resource
// @Description Download a resource from a bucket by its hash
// @Tags resources
// @Produce application/octet-stream
// @Security BearerAuth
// @Param bucket path string true "Bucket ID"
// @Param hash path string true "Resource hash (SHA-256)"
// @Success 200 {file} binary
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /resources/{bucket}/{hash} [get]
func (c *ResourceController) Download(ctx echo.Context) error {
	clientID := middleware.GetClientID(ctx)
	bucketID := ctx.Param("bucket")
	hash := ctx.Param("hash")

	reader, resource, err := c.service.Download(ctx.Request().Context(), clientID, bucketID, hash)
	if err != nil {
		if errors.Is(err, bucketrepo.ErrBucketNotFound) {
			return response.NotFound(ctx, "bucket not found")
		}
		if errors.Is(err, repository.ErrResourceNotFound) {
			return response.NotFound(ctx, "resource not found")
		}
		return response.InternalError(ctx, err.Error())
	}
	defer reader.Close()

	ctx.Response().Header().Set("X-Resource-Hash", resource.Hash)
	ctx.Response().Header().Set("Content-Length", fmt.Sprintf("%d", resource.Size))

	return ctx.Stream(http.StatusOK, resource.ContentType, reader)
}

// Head godoc
// @Summary Get resource metadata
// @Description Get metadata of a resource without downloading the content
// @Tags resources
// @Produce json
// @Security BearerAuth
// @Param bucket path string true "Bucket ID"
// @Param hash path string true "Resource hash (SHA-256)"
// @Success 200 {header} string X-Resource-Hash "Resource hash"
// @Success 200 {header} string Content-Type "Resource content type"
// @Success 200 {header} string Content-Length "Resource size in bytes"
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /resources/{bucket}/{hash} [head]
func (c *ResourceController) Head(ctx echo.Context) error {
	clientID := middleware.GetClientID(ctx)
	bucketID := ctx.Param("bucket")
	hash := ctx.Param("hash")

	resource, err := c.service.Get(ctx.Request().Context(), clientID, bucketID, hash)
	if err != nil {
		if errors.Is(err, bucketrepo.ErrBucketNotFound) {
			return response.NotFound(ctx, "bucket not found")
		}
		if errors.Is(err, repository.ErrResourceNotFound) {
			return response.NotFound(ctx, "resource not found")
		}
		return response.InternalError(ctx, err.Error())
	}

	ctx.Response().Header().Set("X-Resource-Hash", resource.Hash)
	ctx.Response().Header().Set("Content-Type", resource.ContentType)
	ctx.Response().Header().Set("Content-Length", fmt.Sprintf("%d", resource.Size))

	return ctx.NoContent(http.StatusOK)
}

// List godoc
// @Summary List resources in a bucket
// @Description List all resources in a bucket
// @Tags resources
// @Produce json
// @Security BearerAuth
// @Param bucket path string true "Bucket ID"
// @Success 200 {object} response.Response{data=dto.ResourceListResponse}
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /resources/{bucket} [get]
func (c *ResourceController) List(ctx echo.Context) error {
	clientID := middleware.GetClientID(ctx)
	bucketID := ctx.Param("bucket")

	resources, err := c.service.List(ctx.Request().Context(), clientID, bucketID)
	if err != nil {
		if errors.Is(err, bucketrepo.ErrBucketNotFound) {
			return response.NotFound(ctx, "bucket not found")
		}
		return response.InternalError(ctx, err.Error())
	}

	return response.Success(ctx, resources)
}

// Delete godoc
// @Summary Delete a resource
// @Description Delete a resource from a bucket by its hash
// @Tags resources
// @Produce json
// @Security BearerAuth
// @Param bucket path string true "Bucket ID"
// @Param hash path string true "Resource hash (SHA-256)"
// @Success 204
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /resources/{bucket}/{hash} [delete]
func (c *ResourceController) Delete(ctx echo.Context) error {
	clientID := middleware.GetClientID(ctx)
	bucketID := ctx.Param("bucket")
	hash := ctx.Param("hash")

	if err := c.service.Delete(ctx.Request().Context(), clientID, bucketID, hash); err != nil {
		if errors.Is(err, bucketrepo.ErrBucketNotFound) {
			return response.NotFound(ctx, "bucket not found")
		}
		if errors.Is(err, repository.ErrResourceNotFound) {
			return response.NotFound(ctx, "resource not found")
		}
		return response.InternalError(ctx, err.Error())
	}

	return response.NoContent(ctx)
}
