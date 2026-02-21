package config

import (
	"log/slog"
	"net/url"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	DBUrl       string
	Environment string
	Port        string
	JWTSecret   string
	JWTExpiry   time.Duration
}

// Load loads configuration from environment variables.
// It attempts to load from .env file if not in production.
// If logger is non-nil, .env load warnings are logged via slog; otherwise no warning is logged.
func Load(logger *slog.Logger) (*Config, error) {
	env := os.Getenv("GO_ENV")
	if env == "" {
		env = "development"
	}

	// Load .env file if not in production
	// We don't return error here because in production .env might not exist
	// and we rely on system environment variables
	if env != "production" {
		if err := godotenv.Load(); err != nil {
			if logger != nil {
				logger.Warn(".env file not found or couldn't be loaded", "err", err)
			}
		}
	}

	jwtExpiry := 24 * time.Hour
	if s := os.Getenv("JWT_EXPIRY"); s != "" {
		if d, err := time.ParseDuration(s); err == nil {
			jwtExpiry = d
		}
	}

	cfg := &Config{
		Environment: env,
		DBUrl:       os.Getenv("DATABASE_URL"),
		Port:        os.Getenv("PORT"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		JWTExpiry:   jwtExpiry,
	}

	// Set defaults
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	if cfg.DBUrl == "" {
		cfg.DBUrl = "postgres://postgres:postgres@localhost:5432/multitrackticketing?sslmode=disable"
	} else {
		// Ensure sslmode is set. Many hosted DBs (e.g. Sevalla) don't enable SSL;
		// lib/pq defaults to sslmode=prefer which fails with "SSL is not enabled on the server".
		cfg.DBUrl = setDefaultSSLMode(cfg.DBUrl, "disable")
	}

	return cfg, nil
}

// setDefaultSSLMode adds sslmode=defaultMode to the Postgres URL if no sslmode is set.
func setDefaultSSLMode(dbURL, defaultMode string) string {
	u, err := url.Parse(dbURL)
	if err != nil {
		return dbURL
	}
	q := u.Query()
	if q.Get("sslmode") != "" {
		return dbURL
	}
	q.Set("sslmode", defaultMode)
	u.RawQuery = q.Encode()
	return u.String()
}
