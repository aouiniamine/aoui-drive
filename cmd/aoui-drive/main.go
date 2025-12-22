package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/aouiniamine/aoui-drive/docs"

	"github.com/aouiniamine/aoui-drive/internal/cache"
	"github.com/aouiniamine/aoui-drive/internal/config"
	"github.com/aouiniamine/aoui-drive/internal/database"
	"github.com/aouiniamine/aoui-drive/internal/features/auth"
	"github.com/aouiniamine/aoui-drive/internal/features/bucket"
	"github.com/aouiniamine/aoui-drive/internal/features/health"
	"github.com/aouiniamine/aoui-drive/internal/features/resource"
	"github.com/aouiniamine/aoui-drive/internal/middleware"
	"github.com/aouiniamine/aoui-drive/internal/server"
	"github.com/joho/godotenv"
	echoSwagger "github.com/swaggo/echo-swagger"
)

// @title AOUI Drive API
// @version 1.0
// @description MinIO-like Object Storage API
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@aoui-drive.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter your bearer token in the format: Bearer <token>

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	cfg := config.Load()

	db, err := database.New(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	redisCache, err := cache.NewRedis(cache.RedisConfig{
		Host:     cfg.Redis.Host,
		Port:     cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisCache.Close()

	srv := server.New(cfg, db, redisCache)

	srv.Echo().GET("/swagger/*", echoSwagger.WrapHandler)

	healthFeature := health.New(db, redisCache)
	healthFeature.RegisterRoutes(srv.Echo())

	authFeature := auth.New(db, cfg.JWTSecret)
	authFeature.RegisterRoutes(srv.Echo())

	bucketFeature := bucket.New(db, cfg.Storage.Path)
	bucketGroup := srv.Echo().Group("/buckets", middleware.Auth(authFeature.Service))
	bucketFeature.RegisterRoutes(bucketGroup)

	resourceFeature := resource.New(db, bucketFeature.Repository, cfg.Storage.Path, cfg.Storage.PublicURL)
	resourceGroup := srv.Echo().Group("/resources", middleware.Auth(authFeature.Service))
	resourceFeature.RegisterRoutes(resourceGroup)

	// Serve public files with caching headers
	publicPath := cfg.Storage.Path + "/public"
	srv.Echo().Static("/public", publicPath)

	go func() {
		log.Printf("Starting server on %s:%s", cfg.Server.Host, cfg.Server.Port)
		if err := srv.Start(); err != nil {
			log.Printf("Server stopped: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}
