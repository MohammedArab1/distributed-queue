// Package config reads settings from environment variables. Every setting has
// a development-friendly default so the binaries run with no setup.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"
)

// defaultDatabaseURL points at the Postgres started by `make up`.
const defaultDatabaseURL = "postgres://jobqueue:jobqueue@localhost:5433/jobqueue?sslmode=disable"

// Server holds the settings used by cmd/server. Later steps add fields.
type Server struct {
	DatabaseURL string
	LogLevel    slog.Level
}

// LoadServer reads the server's settings from the environment.
func LoadServer() (Server, error) {
	logLevel, err := getLogLevel("JQ_LOG_LEVEL", slog.LevelInfo)
	if err != nil {
		return Server{}, fmt.Errorf("JQ_LOG_LEVEL: %w", err)
	}
	return Server{
		DatabaseURL: getString("JQ_DATABASE_URL", defaultDatabaseURL),
		LogLevel:    logLevel,
	}, nil
}

// getString returns the value of key, or def when it is unset or empty.
func getString(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// getInt returns the integer in key, or def when it is unset. The caller adds
// the variable name to the error.
func getInt(key string, def int) (int, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return def, nil
	}
	return strconv.Atoi(raw)
}

// getDuration returns the Go duration in key (like "30s"), or def when it is
// unset. The caller adds the variable name to the error.
func getDuration(key string, def time.Duration) (time.Duration, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return def, nil
	}
	return time.ParseDuration(raw)
}

// getLogLevel returns the slog level named by key (debug, info, warn or error,
// any case), or def when it is unset.
func getLogLevel(key string, def slog.Level) (slog.Level, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return def, nil
	}
	var level slog.Level
	if err := level.UnmarshalText([]byte(raw)); err != nil {
		return 0, err
	}
	return level, nil
}
