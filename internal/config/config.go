package config

import (
	"os"
	"strconv"
)

type Config struct {
	Server      ServerConfig
	Database    DatabaseConfig
	Redis       RedisConfig
	Storage     StorageConfig
	JWTSecret   string
	Env         string
}

type StorageConfig struct {
	Path      string
	PublicURL string
}

type ServerConfig struct {
	Host string
	Port string
}

type DatabaseConfig struct {
	Path string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Host: getEnv("HOST", "0.0.0.0"),
			Port: getEnv("PORT", "8080"),
		},
		Database: DatabaseConfig{
			Path: getEnv("DATABASE_PATH", "./data/aoui-drive.db"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		Storage: StorageConfig{
			Path:      getEnv("STORAGE_PATH", "./data/storage"),
			PublicURL: getEnv("PUBLIC_URL", ""),
		},
		JWTSecret: getEnv("JWT_SECRET", "change-me-in-production"),
		Env:       getEnv("ENV", "development"),
	}
}

func (c *Config) IsDevelopment() bool {
	return c.Env == "development"
}

func (c *Config) IsProduction() bool {
	return c.Env == "production"
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
