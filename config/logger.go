package config

import (
	"log/slog"
	"os"
)

// NewLogger returns a slog.Logger configured from GO_ENV and LOG_LEVEL.
// Production uses JSON handler; otherwise text handler.
// LOG_LEVEL may be: debug, info, warn, error (default: info).
func NewLogger() *slog.Logger {
	env := os.Getenv("GO_ENV")
	if env == "" {
		env = "development"
	}
	level := slog.LevelInfo
	if s := os.Getenv("LOG_LEVEL"); s != "" {
		switch s {
		case "debug":
			level = slog.LevelDebug
		case "info":
			level = slog.LevelInfo
		case "warn":
			level = slog.LevelWarn
		case "error":
			level = slog.LevelError
		}
	}
	if env == "production" {
		return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}
