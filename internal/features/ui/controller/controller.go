package controller

import (
	"net/http"
	"strconv"

	"github.com/aouiniamine/aoui-drive/internal/features/auth/dto"
	authservice "github.com/aouiniamine/aoui-drive/internal/features/auth/service"
	bucketservice "github.com/aouiniamine/aoui-drive/internal/features/bucket/service"
	resourceservice "github.com/aouiniamine/aoui-drive/internal/features/resource/service"
	"github.com/labstack/echo/v4"
)

const (
	sessionCookieName = "session"
	defaultPerPage    = 20
)

type UIController struct {
	authSvc     authservice.AuthService
	bucketSvc   bucketservice.BucketService
	resourceSvc resourceservice.ResourceService
	publicURL   string
}

func New(authSvc authservice.AuthService, bucketSvc bucketservice.BucketService, resourceSvc resourceservice.ResourceService, publicURL string) *UIController {
	return &UIController{
		authSvc:     authSvc,
		bucketSvc:   bucketSvc,
		resourceSvc: resourceSvc,
		publicURL:   publicURL,
	}
}

// AuthMiddleware checks for valid JWT in cookie
func (c *UIController) AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		cookie, err := ctx.Cookie(sessionCookieName)
		if err != nil || cookie.Value == "" {
			return ctx.Redirect(http.StatusFound, "/ui/login?error=Please+login+to+continue")
		}

		claims, err := c.authSvc.ValidateToken(cookie.Value)
		if err != nil {
			// Clear invalid cookie
			c.clearSessionCookie(ctx)
			return ctx.Redirect(http.StatusFound, "/ui/login?error=Session+expired")
		}

		// Store client ID in context
		ctx.Set("client_id", claims.ClientID)
		return next(ctx)
	}
}

func (c *UIController) RedirectToLogin(ctx echo.Context) error {
	return ctx.Redirect(http.StatusFound, "/ui/login")
}

func (c *UIController) LoginPage(ctx echo.Context) error {
	// Check if already logged in
	cookie, err := ctx.Cookie(sessionCookieName)
	if err == nil && cookie.Value != "" {
		if _, err := c.authSvc.ValidateToken(cookie.Value); err == nil {
			return ctx.Redirect(http.StatusFound, "/ui/buckets")
		}
	}

	return ctx.Render(http.StatusOK, "login.html", map[string]interface{}{
		"Error": ctx.QueryParam("error"),
	})
}

func (c *UIController) Login(ctx echo.Context) error {
	accessKey := ctx.FormValue("access_key")
	secretKey := ctx.FormValue("secret_key")

	if accessKey == "" || secretKey == "" {
		return ctx.Redirect(http.StatusFound, "/ui/login?error=Access+key+and+secret+key+are+required")
	}

	tokenResp, err := c.authSvc.Login(ctx.Request().Context(), dto.LoginRequest{
		AccessKey: accessKey,
		SecretKey: secretKey,
	})
	if err != nil {
		return ctx.Redirect(http.StatusFound, "/ui/login?error=Invalid+credentials")
	}

	// Set session cookie
	ctx.SetCookie(&http.Cookie{
		Name:     sessionCookieName,
		Value:    tokenResp.AccessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   ctx.Request().TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400, // 24 hours in seconds
	})

	return ctx.Redirect(http.StatusSeeOther, "/ui/buckets")
}

func (c *UIController) Logout(ctx echo.Context) error {
	c.clearSessionCookie(ctx)
	return ctx.Redirect(http.StatusFound, "/ui/login")
}

func (c *UIController) BucketsPage(ctx echo.Context) error {
	clientID := ctx.Get("client_id").(string)

	buckets, err := c.bucketSvc.List(ctx.Request().Context(), clientID)
	if err != nil {
		return ctx.Render(http.StatusInternalServerError, "buckets.html", map[string]interface{}{
			"Error": "Failed to load buckets",
		})
	}

	return ctx.Render(http.StatusOK, "buckets.html", map[string]interface{}{
		"Buckets": buckets.Buckets,
	})
}

func (c *UIController) BucketPage(ctx echo.Context) error {
	clientID := ctx.Get("client_id").(string)
	bucketID := ctx.Param("id")

	bucket, err := c.bucketSvc.Get(ctx.Request().Context(), clientID, bucketID)
	if err != nil {
		return ctx.Render(http.StatusNotFound, "bucket.html", map[string]interface{}{
			"Error": "Bucket not found",
		})
	}

	page, perPage := c.getPagination(ctx)

	resources, err := c.resourceSvc.List(ctx.Request().Context(), clientID, bucketID)
	if err != nil {
		return ctx.Render(http.StatusInternalServerError, "bucket.html", map[string]interface{}{
			"Bucket": bucket,
			"Error":  "Failed to load resources",
		})
	}

	// Calculate pagination
	total := len(resources.Resources)
	totalPages := (total + perPage - 1) / perPage
	if page > totalPages && totalPages > 0 {
		page = totalPages
	}

	start := (page - 1) * perPage
	end := start + perPage
	if end > total {
		end = total
	}

	var paginatedResources []interface{}
	if start < total {
		for _, r := range resources.Resources[start:end] {
			paginatedResources = append(paginatedResources, r)
		}
	}

	data := map[string]interface{}{
		"Bucket":     bucket,
		"Resources":  paginatedResources,
		"Page":       page,
		"PerPage":    perPage,
		"Total":      total,
		"TotalPages": totalPages,
		"PublicURL":  c.publicURL,
	}

	return ctx.Render(http.StatusOK, "bucket.html", data)
}

func (c *UIController) ResourcesPartial(ctx echo.Context) error {
	clientID := ctx.Get("client_id").(string)
	bucketID := ctx.Param("id")

	bucket, err := c.bucketSvc.Get(ctx.Request().Context(), clientID, bucketID)
	if err != nil {
		return ctx.HTML(http.StatusNotFound, "<p class='text-red-500'>Bucket not found</p>")
	}

	page, perPage := c.getPagination(ctx)

	resources, err := c.resourceSvc.List(ctx.Request().Context(), clientID, bucketID)
	if err != nil {
		return ctx.HTML(http.StatusInternalServerError, "<p class='text-red-500'>Failed to load resources</p>")
	}

	// Calculate pagination
	total := len(resources.Resources)
	totalPages := (total + perPage - 1) / perPage
	if page > totalPages && totalPages > 0 {
		page = totalPages
	}

	start := (page - 1) * perPage
	end := start + perPage
	if end > total {
		end = total
	}

	var paginatedResources []interface{}
	if start < total {
		for _, r := range resources.Resources[start:end] {
			paginatedResources = append(paginatedResources, r)
		}
	}

	data := map[string]interface{}{
		"Bucket":     bucket,
		"Resources":  paginatedResources,
		"Page":       page,
		"PerPage":    perPage,
		"Total":      total,
		"TotalPages": totalPages,
		"PublicURL":  c.publicURL,
	}

	return ctx.Render(http.StatusOK, "resource-list.html", data)
}

func (c *UIController) DeleteResource(ctx echo.Context) error {
	clientID := ctx.Get("client_id").(string)
	bucketID := ctx.Param("id")
	hash := ctx.Param("hash")

	err := c.resourceSvc.Delete(ctx.Request().Context(), clientID, bucketID, hash)
	if err != nil {
		return ctx.HTML(http.StatusInternalServerError, "<p class='text-red-500'>Failed to delete resource</p>")
	}

	// Return empty response - HTMX will remove the element
	ctx.Response().Header().Set("HX-Trigger", "resourceDeleted")
	return ctx.NoContent(http.StatusOK)
}

func (c *UIController) ViewResource(ctx echo.Context) error {
	clientID := ctx.Get("client_id").(string)
	bucketID := ctx.Param("id")
	hash := ctx.Param("hash")

	file, resource, err := c.resourceSvc.Download(ctx.Request().Context(), clientID, bucketID, hash)
	if err != nil {
		return ctx.String(http.StatusNotFound, "Resource not found")
	}
	defer file.Close()

	ctx.Response().Header().Set("Content-Type", resource.ContentType)
	ctx.Response().Header().Set("Content-Length", strconv.FormatInt(resource.Size, 10))
	ctx.Response().Header().Set("Cache-Control", "private, max-age=3600")

	return ctx.Stream(http.StatusOK, resource.ContentType, file)
}

func (c *UIController) DownloadResource(ctx echo.Context) error {
	clientID := ctx.Get("client_id").(string)
	bucketID := ctx.Param("id")
	hash := ctx.Param("hash")

	file, resource, err := c.resourceSvc.Download(ctx.Request().Context(), clientID, bucketID, hash)
	if err != nil {
		return ctx.String(http.StatusNotFound, "Resource not found")
	}
	defer file.Close()

	filename := resource.Hash + resource.Extension
	ctx.Response().Header().Set("Content-Type", resource.ContentType)
	ctx.Response().Header().Set("Content-Length", strconv.FormatInt(resource.Size, 10))
	ctx.Response().Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")

	return ctx.Stream(http.StatusOK, resource.ContentType, file)
}

func (c *UIController) UploadResources(ctx echo.Context) error {
	clientID := ctx.Get("client_id").(string)
	bucketID := ctx.Param("id")

	form, err := ctx.MultipartForm()
	if err != nil {
		return ctx.HTML(http.StatusBadRequest, `<div class="text-red-600 text-sm">Failed to parse upload</div>`)
	}

	files := form.File["files"]
	if len(files) == 0 {
		return ctx.HTML(http.StatusBadRequest, `<div class="text-red-600 text-sm">No files selected</div>`)
	}

	var uploaded int
	var errors []string

	for _, file := range files {
		_, err := c.resourceSvc.UploadFile(ctx.Request().Context(), clientID, bucketID, file)
		if err != nil {
			errors = append(errors, file.Filename+": "+err.Error())
		} else {
			uploaded++
		}
	}

	// Trigger refresh of resource list
	ctx.Response().Header().Set("HX-Trigger", "resourceUploaded")

	if len(errors) > 0 {
		return ctx.HTML(http.StatusOK, `<div class="text-yellow-600 text-sm">`+strconv.Itoa(uploaded)+` files uploaded, `+strconv.Itoa(len(errors))+` failed</div>`)
	}

	return ctx.HTML(http.StatusOK, `<div class="text-green-600 text-sm">`+strconv.Itoa(uploaded)+` files uploaded successfully</div>`)
}

func (c *UIController) clearSessionCookie(ctx echo.Context) {
	cookie := &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	}
	ctx.SetCookie(cookie)
}

func (c *UIController) getPagination(ctx echo.Context) (page, perPage int) {
	page = 1
	perPage = defaultPerPage

	if p := ctx.QueryParam("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if pp := ctx.QueryParam("per_page"); pp != "" {
		if parsed, err := strconv.Atoi(pp); err == nil && parsed > 0 && parsed <= 100 {
			perPage = parsed
		}
	}

	return
}
