// Package testutil provides helpers for tests that use real infrastructure.
package testutil

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/MohammedArab1/jobqueue/internal/store"
)

const defaultTestDatabaseURL = "postgres://jobqueue:jobqueue@localhost:5433/postgres?sslmode=disable"

// NewTestDB creates a fresh database, applies all migrations, and returns a
// pool connected to it. The database is dropped when the test finishes unless
// JQ_KEEP_TEST_DB=1, in which case its name is logged.
func NewTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	adminURL := os.Getenv("JQ_TEST_DATABASE_URL")
	if adminURL == "" {
		adminURL = defaultTestDatabaseURL
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	admin, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		t.Fatalf("connect to test postgres: %v (is Postgres running? try `make up`)", err)
	}
	defer admin.Close(ctx)

	name := "jq_test_" + randomHex(t, 8)
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
		t.Fatalf("create test database %s: %v", name, err)
	}

	testURL, err := databaseURLFor(adminURL, name)
	if err != nil {
		cleanupDatabase(t, adminURL, name)
		t.Fatalf("build test database URL: %v", err)
	}
	if err := store.Migrate(ctx, testURL, "up"); err != nil {
		cleanupDatabase(t, adminURL, name)
		t.Fatalf("migrate test database %s: %v", name, err)
	}

	pool, err := pgxpool.New(context.Background(), testURL)
	if err != nil {
		cleanupDatabase(t, adminURL, name)
		t.Fatalf("open test database pool: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		cleanupDatabase(t, adminURL, name)
		t.Fatalf("ping test database %s: %v", name, err)
	}

	t.Cleanup(func() {
		pool.Close()
		cleanupDatabase(t, adminURL, name)
	})
	return pool
}

// cleanupDatabase drops the test database, or logs its name when the developer
// asked to keep it for debugging.
func cleanupDatabase(t *testing.T, adminURL, name string) {
	t.Helper()

	if os.Getenv("JQ_KEEP_TEST_DB") == "1" {
		t.Logf("keeping test database %s", name)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	admin, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		t.Errorf("connect to drop test database %s: %v", name, err)
		return
	}
	defer admin.Close(ctx)

	dropSQL := "DROP DATABASE " + pgx.Identifier{name}.Sanitize() + " WITH (FORCE)"
	if _, err := admin.Exec(ctx, dropSQL); err != nil {
		t.Errorf("drop test database %s: %v", name, err)
	}
}

// databaseURLFor returns adminURL with its database path replaced by name.
func databaseURLFor(adminURL, name string) (string, error) {
	u, err := url.Parse(adminURL)
	if err != nil {
		return "", fmt.Errorf("parse test database URL: %w", err)
	}
	u.Path = "/" + name
	return u.String(), nil
}

// randomHex returns size cryptographically random bytes as hex.
func randomHex(t *testing.T, size int) string {
	t.Helper()

	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("generate test database name: %v", err)
	}
	return hex.EncodeToString(b)
}
