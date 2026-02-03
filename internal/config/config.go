package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server    Server
	Database  Database
	JWT       JWT
	UploadPath string
	BaseURL   string
}

type Server struct {
	Port string
}

type Database struct {
	DSN string
}

type JWT struct {
	AccessSecret  string
	RefreshSecret string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

func Load() *Config {
	_ = godotenv.Load()

	accessTTL := 15 * time.Minute
	refreshTTL := 30 * 24 * time.Hour // 30 дней
	if v := os.Getenv("JWT_ACCESS_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			accessTTL = d
		}
	}
	if v := os.Getenv("JWT_REFRESH_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			refreshTTL = d
		}
	}

	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		host := getEnv("DB_HOST", "localhost")
		port := getEnv("DB_PORT", "5432")
		user := getEnv("DB_USER", "postgres")
		pass := getEnv("DB_PASSWORD", "postgres")
		dbname := getEnv("DB_NAME", "mozgoemka")
		sslmode := getEnv("DB_SSLMODE", "disable")
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, pass, host, port, dbname, sslmode)
	}

	uploadPath := getEnv("UPLOAD_PATH", "./uploads")
	uploadPath, _ = filepath.Abs(uploadPath)
	baseURL := getEnv("SERVER_BASE_URL", "http://localhost:8080")
	if port := getEnv("SERVER_PORT", "8080"); port != "" && baseURL == "http://localhost:8080" {
		baseURL = "http://localhost:" + port
	}

	port := getEnv("SERVER_PORT", getEnv("PORT", "8080"))
	return &Config{
		Server: Server{
			Port: port,
		},
		Database: Database{DSN: dsn},
		JWT: JWT{
			AccessSecret:  getEnv("JWT_ACCESS_SECRET", "change-me-access-secret"),
			RefreshSecret: getEnv("JWT_REFRESH_SECRET", "change-me-refresh-secret"),
			AccessTTL:     accessTTL,
			RefreshTTL:    refreshTTL,
		},
		UploadPath: uploadPath,
		BaseURL:    baseURL,
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
