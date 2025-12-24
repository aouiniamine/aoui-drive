package controller

import (
	"net/http"
	"strconv"

	"github.com/aouiniamine/aoui-drive/internal/features/auth/dto"
	authservice "github.com/aouiniamine/aoui-drive/internal/features/auth/service"
	bucketservice "github.com/aouiniamine/aoui-drive/internal/features/bucket/service"
	resourceservice "github.com/aouiniamine/aoui-drive/internal/features/resource/service"
	webhookdto "github.com/aouiniamine/aoui-drive/internal/features/webhook/dto"
	webhookservice "github.com/aouiniamine/aoui-drive/internal/features/webhook/service"
	"github.com/aouiniamine/aoui-drive/internal/middleware"
	"github.com/labstack/echo/v4"
)

const (
	defaultPerPage = 20
)

type UIController struct {
	authSvc     authservice.AuthService
	bucketSvc   bucketservice.BucketService
	resourceSvc resourceservice.ResourceService
	webhookSvc  webhookservice.WebhookService
	publicURL   string
}

func New(authSvc authservice.AuthService, bucketSvc bucketservice.BucketService, resourceSvc resourceservice.ResourceService, webhookSvc webhookservice.WebhookService, publicURL string) *UIController {
	return &UIController{
		authSvc:     authSvc,
		bucketSvc:   bucketSvc,
		resourceSvc: resourceSvc,
		webhookSvc:  webhookSvc,
		publicURL:   publicURL,
	}
}

func (c *UIController) RedirectToLogin(ctx echo.Context) error {
	return ctx.Redirect(http.StatusFound, "/ui/login")
}

func (c *UIController) LoginPage(ctx echo.Context) error {
	// Check if already logged in
	cookie, err := ctx.Cookie(middleware.SessionCookieName)
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
		Name:     middleware.SessionCookieName,
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
	clientID := middleware.GetClientID(ctx)

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
	clientID := middleware.GetClientID(ctx)
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
	clientID := middleware.GetClientID(ctx)
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
	clientID := middleware.GetClientID(ctx)
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
	clientID := middleware.GetClientID(ctx)
	bucketID := ctx.Param("id")
	hash := ctx.Param("hash")

	file, resource, err := c.resourceSvc.Download(ctx.Request().Context(), clientID, bucketID, hash)
	if err != nil {
		return ctx.String(http.StatusNotFound, "Resource not found")
	}
	defer file.Close()

	ctx.Response().Header().Set("Content-Type", resource.ContentType)
	ctx.Response().Header().Set("Cache-Control", "private, max-age=3600")

	return ctx.Stream(http.StatusOK, resource.ContentType, file)
}

func (c *UIController) DownloadResource(ctx echo.Context) error {
	clientID := middleware.GetClientID(ctx)
	bucketID := ctx.Param("id")
	hash := ctx.Param("hash")

	file, resource, err := c.resourceSvc.Download(ctx.Request().Context(), clientID, bucketID, hash)
	if err != nil {
		return ctx.String(http.StatusNotFound, "Resource not found")
	}
	defer file.Close()

	filename := resource.Hash + resource.Extension
	ctx.Response().Header().Set("Content-Type", resource.ContentType)
	ctx.Response().Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")

	return ctx.Stream(http.StatusOK, resource.ContentType, file)
}

func (c *UIController) UploadResources(ctx echo.Context) error {
	clientID := middleware.GetClientID(ctx)
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
		_, err := c.resourceSvc.UploadFile(ctx.Request().Context(), clientID, bucketID, file, nil)
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
		Name:     middleware.SessionCookieName,
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

// Webhook UI handlers

func (c *UIController) WebhooksPage(ctx echo.Context) error {
	clientID := middleware.GetClientID(ctx)
	bucketID := ctx.Param("id")

	bucket, err := c.bucketSvc.Get(ctx.Request().Context(), clientID, bucketID)
	if err != nil {
		return ctx.Redirect(http.StatusFound, "/ui/buckets")
	}

	webhooks, _ := c.webhookSvc.ListURLs(ctx.Request().Context(), clientID, bucketID)
	var webhookList []webhookdto.WebhookURLResponse
	if webhooks != nil {
		webhookList = webhooks.Webhooks
	}

	return ctx.Render(http.StatusOK, "webhooks-page.html", map[string]interface{}{
		"Bucket":   bucket,
		"Webhooks": webhookList,
	})
}

func (c *UIController) WebhooksListPartial(ctx echo.Context) error {
	clientID := middleware.GetClientID(ctx)
	bucketID := ctx.Param("id")

	bucket, err := c.bucketSvc.Get(ctx.Request().Context(), clientID, bucketID)
	if err != nil {
		return ctx.HTML(http.StatusNotFound, "<p class='text-red-500'>Bucket not found</p>")
	}

	webhooks, err := c.webhookSvc.ListURLs(ctx.Request().Context(), clientID, bucketID)
	if err != nil {
		return ctx.HTML(http.StatusInternalServerError, "<p class='text-red-500'>Failed to load webhooks</p>")
	}

	return ctx.Render(http.StatusOK, "webhooks-list.html", map[string]interface{}{
		"Bucket":   bucket,
		"Webhooks": webhooks.Webhooks,
	})
}

func (c *UIController) CreateWebhook(ctx echo.Context) error {
	clientID := middleware.GetClientID(ctx)
	bucketID := ctx.Param("id")

	url := ctx.FormValue("url")
	eventType := ctx.FormValue("event_type")
	isActive := ctx.FormValue("is_active") == "on"

	if url == "" || eventType == "" {
		return ctx.HTML(http.StatusBadRequest, `<div class="text-red-600 text-sm">URL and event type are required</div>`)
	}

	_, err := c.webhookSvc.CreateURL(ctx.Request().Context(), clientID, bucketID, webhookdto.CreateWebhookURLRequest{
		URL:       url,
		EventType: eventType,
		IsActive:  isActive,
	})
	if err != nil {
		return ctx.HTML(http.StatusBadRequest, `<div class="text-red-600 text-sm">`+err.Error()+`</div>`)
	}

	// Trigger refresh of webhook list
	ctx.Response().Header().Set("HX-Trigger", "webhookCreated")
	return ctx.HTML(http.StatusOK, `<div class="text-green-600 text-sm">Webhook created successfully</div>`)
}

func (c *UIController) DeleteWebhook(ctx echo.Context) error {
	clientID := middleware.GetClientID(ctx)
	bucketID := ctx.Param("id")
	webhookID := ctx.Param("webhookId")

	err := c.webhookSvc.DeleteURL(ctx.Request().Context(), clientID, bucketID, webhookID)
	if err != nil {
		return ctx.HTML(http.StatusInternalServerError, "<p class='text-red-500'>Failed to delete webhook</p>")
	}

	// Return empty response - HTMX will remove the element
	ctx.Response().Header().Set("HX-Trigger", "webhookDeleted")
	return ctx.NoContent(http.StatusOK)
}

func (c *UIController) CreateWebhookHeader(ctx echo.Context) error {
	clientID := middleware.GetClientID(ctx)
	bucketID := ctx.Param("id")
	webhookID := ctx.Param("webhookId")

	headerName := ctx.FormValue("header_name")
	headerValue := ctx.FormValue("header_value")

	if headerName == "" || headerValue == "" {
		return ctx.HTML(http.StatusBadRequest, `<div class="text-red-600 text-sm">Header name and value are required</div>`)
	}

	_, err := c.webhookSvc.CreateHeader(ctx.Request().Context(), clientID, bucketID, webhookID, webhookdto.CreateHeaderRequest{
		Name:  headerName,
		Value: headerValue,
	})
	if err != nil {
		return ctx.HTML(http.StatusBadRequest, `<div class="text-red-600 text-sm">`+err.Error()+`</div>`)
	}

	// Return refreshed webhooks list
	return c.WebhooksListPartial(ctx)
}

func (c *UIController) DeleteWebhookHeader(ctx echo.Context) error {
	clientID := middleware.GetClientID(ctx)
	bucketID := ctx.Param("id")
	webhookID := ctx.Param("webhookId")
	headerID := ctx.Param("headerId")

	err := c.webhookSvc.DeleteHeader(ctx.Request().Context(), clientID, bucketID, webhookID, headerID)
	if err != nil {
		return ctx.HTML(http.StatusInternalServerError, "<p class='text-red-500'>Failed to delete header</p>")
	}

	// Return refreshed webhooks list
	return c.WebhooksListPartial(ctx)
}
