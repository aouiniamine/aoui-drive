package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aouiniamine/aoui-drive/internal/cache"
	"github.com/aouiniamine/aoui-drive/internal/config"
	"github.com/aouiniamine/aoui-drive/internal/database"
	"github.com/aouiniamine/aoui-drive/internal/features/auth"
	"github.com/aouiniamine/aoui-drive/internal/features/health"
	"github.com/aouiniamine/aoui-drive/internal/server"
	"github.com/joho/godotenv"
)

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

	healthFeature := health.New(db, redisCache)
	healthFeature.RegisterRoutes(srv.Echo())

	authFeature := auth.New(db, cfg.JWTSecret)
	authFeature.RegisterRoutes(srv.Echo())

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
