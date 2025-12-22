package ui

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"time"

	authservice "github.com/aouiniamine/aoui-drive/internal/features/auth/service"
	bucketservice "github.com/aouiniamine/aoui-drive/internal/features/bucket/service"
	resourceservice "github.com/aouiniamine/aoui-drive/internal/features/resource/service"
	"github.com/aouiniamine/aoui-drive/internal/features/ui/controller"
	"github.com/labstack/echo/v4"
)

//go:embed templates/*
var templatesFS embed.FS

type TemplateRenderer struct {
	templates *template.Template
}

func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

type Feature struct {
	Controller *controller.UIController
}

func New(authSvc authservice.AuthService, bucketSvc bucketservice.BucketService, resourceSvc resourceservice.ResourceService, publicURL string) *Feature {
	ctrl := controller.New(authSvc, bucketSvc, resourceSvc, publicURL)
	return &Feature{
		Controller: ctrl,
	}
}

func (f *Feature) RegisterRoutes(e *echo.Echo) {
	// Parse templates with custom functions
	funcMap := template.FuncMap{
		"formatBytes": formatBytes,
		"formatDate":  formatDate,
		"isImage":     isImage,
		"isPDF":       isPDF,
		"isVideo":     isVideo,
		"isAudio":     isAudio,
		"add":         func(a, b int) int { return a + b },
		"subtract":    func(a, b int) int { return a - b },
	}

	tmpl := template.Must(template.New("").Funcs(funcMap).ParseFS(templatesFS, "templates/*.html", "templates/partials/*.html"))

	e.Renderer = &TemplateRenderer{templates: tmpl}

	// Public routes (no auth required)
	e.GET("/ui", f.Controller.RedirectToLogin)
	e.GET("/ui/login", f.Controller.LoginPage)
	e.POST("/ui/login", f.Controller.Login)

	// Protected routes (auth required via cookie)
	ui := e.Group("/ui")
	ui.Use(f.Controller.AuthMiddleware)

	ui.GET("/logout", f.Controller.Logout)
	ui.GET("/buckets", f.Controller.BucketsPage)
	ui.GET("/buckets/:id", f.Controller.BucketPage)
	ui.GET("/buckets/:id/resources", f.Controller.ResourcesPartial)
	ui.POST("/buckets/:id/upload", f.Controller.UploadResources)
	ui.GET("/buckets/:id/resources/:hash/view", f.Controller.ViewResource)
	ui.GET("/buckets/:id/resources/:hash/download", f.Controller.DownloadResource)
	ui.DELETE("/buckets/:id/resources/:hash", f.Controller.DeleteResource)
}

// Template helper functions
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func formatDate(t time.Time) string {
	return t.Format("Jan 02, 2006 15:04")
}

func isImage(contentType string) bool {
	switch contentType {
	case "image/jpeg", "image/png", "image/gif", "image/webp", "image/svg+xml", "image/bmp":
		return true
	}
	return false
}

func isPDF(contentType string) bool {
	return contentType == "application/pdf"
}

func isVideo(contentType string) bool {
	switch contentType {
	case "video/mp4", "video/webm", "video/ogg":
		return true
	}
	return false
}

func isAudio(contentType string) bool {
	switch contentType {
	case "audio/mpeg", "audio/ogg", "audio/wav", "audio/webm":
		return true
	}
	return false
}
