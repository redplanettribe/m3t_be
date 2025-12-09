package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	DBUrl       string
	Environment string
	Port        string
}

// Load loads configuration from environment variables
// It attempts to load from .env file if not in production
func Load() (*Config, error) {
	env := os.Getenv("GO_ENV")
	if env == "" {
		env = "development"
	}

	// Load .env file if not in production
	// We don't return error here because in production .env might not exist
	// and we rely on system environment variables
	if env != "production" {
		if err := godotenv.Load(); err != nil {
			log.Printf("Warning: .env file not found or couldn't be loaded: %v", err)
		}
	}

	cfg := &Config{
		Environment: env,
		DBUrl:       os.Getenv("DATABASE_URL"),
		Port:        os.Getenv("PORT"),
	}

	// Set defaults
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	// For DBUrl, we let the caller handle empty value or set a default if needed,
	// but based on existing main.go, it might have a default local one.
	// Looking at main.go, it had a hardcoded default. Let's replicate that logic here or let main.go handle it?
	// The request said "Load those environment vars on a config package".
	// So let's put the default here if not found.
	if cfg.DBUrl == "" {
		// Default from previous main.go
		cfg.DBUrl = "postgres://postgres:postgres@localhost:5432/multitrackticketing?sslmode=disable"
	}

	return cfg, nil
}
