package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server                   Server
	Database                 Database
	JWT                      JWT
	RateLimit                RateLimit
	BootstrapAdmin           BootstrapAdmin
	CORSOrigins              []string
	UploadPath               string
	BaseURL                  string
	PasswordResetReturnToken bool
}

// BootstrapAdmin — опциональное автоматическое создание/повышение учётной
// записи администратора при старте сервиса. Если заданы оба поля, при
// каждом запуске пользователь с этим email будет либо создан с ролью admin,
// либо повышен до admin (если уже существовал и не имел этой роли).
type BootstrapAdmin struct {
	Email    string
	Password string
}

// RateLimit — параметры ограничения частоты запросов (ТЗ §4.1).
// «Auth» применяется отдельно к публичным эндпоинтам /auth/* для защиты от
// брутфорса логина и сброса пароля; «Global» — мягкий лимит на весь API.
type RateLimit struct {
	GlobalRPS   float64       // запросов в секунду
	GlobalBurst int           // допустимый пиковый burst
	AuthPerMin  float64       // запросов в минуту
	AuthBurst   int
	IdleTTL     time.Duration // как долго хранить bucket для неактивного IP
}

type Server struct {
	Port        string
	TLSCertFile string // если непустой — сервер запускается через ListenAndServeTLS
	TLSKeyFile  string
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
			Port:        port,
			TLSCertFile: getEnv("TLS_CERT_FILE", ""),
			TLSKeyFile:  getEnv("TLS_KEY_FILE", ""),
		},
		Database: Database{DSN: dsn},
		JWT: JWT{
			AccessSecret:  getEnv("JWT_ACCESS_SECRET", "change-me-access-secret"),
			RefreshSecret: getEnv("JWT_REFRESH_SECRET", "change-me-refresh-secret"),
			AccessTTL:     accessTTL,
			RefreshTTL:    refreshTTL,
		},
		RateLimit: RateLimit{
			GlobalRPS:   getEnvFloat("RATE_LIMIT_GLOBAL_RPS", 100),
			GlobalBurst: getEnvInt("RATE_LIMIT_GLOBAL_BURST", 200),
			AuthPerMin:  getEnvFloat("RATE_LIMIT_AUTH_PER_MIN", 20),
			AuthBurst:   getEnvInt("RATE_LIMIT_AUTH_BURST", 5),
			IdleTTL:     5 * time.Minute,
		},
		BootstrapAdmin: BootstrapAdmin{
			Email:    getEnv("BOOTSTRAP_ADMIN_EMAIL", ""),
			Password: getEnv("BOOTSTRAP_ADMIN_PASSWORD", ""),
		},
		CORSOrigins: parseCSV(getEnv("CORS_ORIGINS", "")),
		UploadPath:               uploadPath,
		BaseURL:                  baseURL,
		PasswordResetReturnToken: getEnv("PASSWORD_RESET_RETURN_TOKEN", "false") == "true",
	}
}

func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
			return n
		}
	}
	return defaultVal
}

// parseCSV режет строку через запятую и обрезает пробелы у каждого элемента.
// Пустые значения отбрасываются. Используется для CORS_ORIGINS и т. п.
func parseCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func getEnvFloat(key string, defaultVal float64) float64 {
	if v := os.Getenv(key); v != "" {
		var f float64
		if _, err := fmt.Sscanf(v, "%f", &f); err == nil {
			return f
		}
	}
	return defaultVal
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
