package config

import (
	"log/slog"
	"testing"
	"time"
)

func TestLoadServer(t *testing.T) {
	tests := []struct {
		name      string
		dbURL     string
		logLevel  string
		wantURL   string
		wantLevel slog.Level
		wantErr   bool
	}{
		{
			name:      "defaults when unset",
			wantURL:   defaultDatabaseURL,
			wantLevel: slog.LevelInfo,
		},
		{
			name:      "values from the environment win",
			dbURL:     "postgres://example:5432/other",
			logLevel:  "warn",
			wantURL:   "postgres://example:5432/other",
			wantLevel: slog.LevelWarn,
		},
		{
			name:     "unparseable log level is an error",
			logLevel: "banana",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("JQ_DATABASE_URL", tt.dbURL)
			t.Setenv("JQ_LOG_LEVEL", tt.logLevel)

			got, err := LoadServer()
			if tt.wantErr {
				if err == nil {
					t.Fatal("LoadServer() error = nil, want an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("LoadServer() error = %v", err)
			}
			if got.DatabaseURL != tt.wantURL {
				t.Errorf("DatabaseURL = %q, want %q", got.DatabaseURL, tt.wantURL)
			}
			if got.LogLevel != tt.wantLevel {
				t.Errorf("LogLevel = %v, want %v", got.LogLevel, tt.wantLevel)
			}
		})
	}
}

func TestGetInt(t *testing.T) {
	tests := []struct {
		name    string
		env     string
		def     int
		want    int
		wantErr bool
	}{
		{name: "unset uses default", env: "", def: 10, want: 10},
		{name: "valid override", env: "42", def: 10, want: 42},
		{name: "invalid value is an error", env: "abc", def: 10, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("JQ_TEST_INT", tt.env)

			got, err := getInt("JQ_TEST_INT", tt.def)
			if tt.wantErr {
				if err == nil {
					t.Fatal("getInt() error = nil, want an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("getInt() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("getInt() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestGetDuration(t *testing.T) {
	tests := []struct {
		name    string
		env     string
		def     time.Duration
		want    time.Duration
		wantErr bool
	}{
		{name: "unset uses default", env: "", def: 30 * time.Second, want: 30 * time.Second},
		{name: "valid override", env: "10s", def: 30 * time.Second, want: 10 * time.Second},
		{name: "invalid duration is an error", env: "soon", def: 30 * time.Second, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("JQ_TEST_DURATION", tt.env)

			got, err := getDuration("JQ_TEST_DURATION", tt.def)
			if tt.wantErr {
				if err == nil {
					t.Fatal("getDuration() error = nil, want an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("getDuration() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("getDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}
