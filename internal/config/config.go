package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	// Server
	Port string
	Env  string

	// Database
	DatabaseURL string

	// Redis
	RedisURL string

	// JWT
	JWTSecret     string
	JWTAccessTTL  time.Duration
	JWTRefreshTTL time.Duration

	// CORS
	AllowedOrigins []string

	// Storage (R2)
	R2AccountID       string
	R2AccessKeyID     string
	R2AccessKeySecret string
	R2BucketName      string
	R2PublicURL       string

	// Email
	ResendAPIKey string

	// Kaspi Payment (DEPRECATED - use RoboKassa)
	KaspiBaseURL    string
	KaspiMerchantID string
	KaspiSecretKey  string

	// RoboKassa Payment
	RoboKassaMerchantLogin string
	RoboKassaPassword1     string
	RoboKassaPassword2     string
	RoboKassaTestMode      bool

	// Payment URLs
	FrontendURL string
	BackendURL  string

	// PhotoStudio
	PhotoStudioBaseURL        string
	PhotoStudioToken          string
	PhotoStudioSyncEnabled    bool
	PhotoStudioTimeoutSeconds int

	// Logging
	LogLevel string
}

func Load() *Config {
	// Load .env file in development
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	return &Config{
		// Server
		Port: getEnv("PORT", "8080"),
		Env:  getEnv("ENV", "development"),

		// Database
		DatabaseURL: getEnv("DATABASE_URL", "postgresql://mwork:mwork_secret@localhost:5432/mwork_dev?sslmode=disable"),

		// Redis
		RedisURL: getEnv("REDIS_URL", "redis://localhost:6379/0"),

		// JWT
		JWTSecret:     getEnv("JWT_SECRET", "super-secret-key-change-me"),
		JWTAccessTTL:  parseDuration(getEnv("JWT_ACCESS_TTL", "15m")),
		JWTRefreshTTL: parseDuration(getEnv("JWT_REFRESH_TTL", "168h")),

		// CORS
		AllowedOrigins: parseStringSlice(getEnv("ALLOWED_ORIGINS", "http://localhost:3000")),

		// Storage
		R2AccountID:       getEnv("R2_ACCOUNT_ID", ""),
		R2AccessKeyID:     getEnv("R2_ACCESS_KEY_ID", ""),
		R2AccessKeySecret: getEnv("R2_ACCESS_KEY_SECRET", ""),
		R2BucketName:      getEnv("R2_BUCKET_NAME", "mwork-uploads"),
		R2PublicURL:       getEnv("R2_PUBLIC_URL", ""),

		// Email
		ResendAPIKey: getEnv("RESEND_API_KEY", ""),

		// Kaspi Payment (DEPRECATED)
		KaspiBaseURL:    getEnv("KASPI_BASE_URL", "https://api.kaspi.kz"),
		KaspiMerchantID: getEnv("KASPI_MERCHANT_ID", ""),
		KaspiSecretKey:  getEnv("KASPI_SECRET_KEY", ""),

		// RoboKassa Payment
		RoboKassaMerchantLogin: getEnv("ROBOKASSA_MERCHANT_LOGIN", ""),
		RoboKassaPassword1:     getEnv("ROBOKASSA_PASSWORD1", ""),
		RoboKassaPassword2:     getEnv("ROBOKASSA_PASSWORD2", ""),
		RoboKassaTestMode:      parseBool(getEnv("ROBOKASSA_TEST_MODE", "false"), false),

		// Payment URLs
		FrontendURL: getEnv("FRONTEND_URL", "http://localhost:3000"),
		BackendURL:  getEnv("BACKEND_URL", "http://localhost:8080"),

		// PhotoStudio
		PhotoStudioBaseURL:        getEnv("PHOTOSTUDIO_BASE_URL", ""),
		PhotoStudioToken:          getEnv("PHOTOSTUDIO_TOKEN", ""),
		PhotoStudioSyncEnabled:    parseBool(getEnv("PHOTOSTUDIO_SYNC_ENABLED", "false"), false),
		PhotoStudioTimeoutSeconds: parseInt(getEnv("PHOTOSTUDIO_TIMEOUT_SECONDS", "10"), 10),

		// Logging
		LogLevel: getEnv("LOG_LEVEL", "debug"),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 15 * time.Minute
	}
	return d
}

func parseBool(s string, defaultValue bool) bool {
	value, err := strconv.ParseBool(s)
	if err != nil {
		return defaultValue
	}
	return value
}

func parseInt(s string, defaultValue int) int {
	value, err := strconv.Atoi(s)
	if err != nil {
		return defaultValue
	}
	return value
}

func parseStringSlice(s string) []string {
	if s == "" {
		return []string{}
	}
	// Simple split by comma
	var result []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			if start < i {
				result = append(result, s[start:i])
			}
			start = i + 1
		}
	}
	return result
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Env == "development"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.Env == "production"
}
