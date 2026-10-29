// Package store holds all SQL for the job queue. Migrations are embedded in
// the binary so every environment runs the same schema code.
package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrate applies schema migrations to the database at databaseURL. command
// must be "up", "down" or "status".
func Migrate(ctx context.Context, databaseURL, command string) error {
	switch command {
	case "up", "down", "status":
	default:
		return fmt.Errorf("unknown migration command %q (want up, down or status)", command)
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	migrations, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		db.Close() // best effort: the original error is more useful
		return fmt.Errorf("open embedded migrations: %w", err)
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations)
	if err != nil {
		db.Close() // best effort: the original error is more useful
		return fmt.Errorf("create migration provider: %w", err)
	}
	defer provider.Close() // closes the database connection too

	switch command {
	case "up":
		return migrateUp(ctx, provider)
	case "down":
		return migrateDown(ctx, provider)
	default:
		return migrateStatus(ctx, provider)
	}
}

// migrateUp applies every pending migration in version order.
func migrateUp(ctx context.Context, provider *goose.Provider) error {
	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	if len(results) == 0 {
		fmt.Println("no migrations to apply")
		return nil
	}
	for _, result := range results {
		fmt.Printf("applied %d %s (%s)\n",
			result.Source.Version,
			filepath.Base(result.Source.Path),
			result.Duration.Round(time.Millisecond),
		)
	}
	return nil
}

// migrateDown rolls back the most recently applied migration.
func migrateDown(ctx context.Context, provider *goose.Provider) error {
	result, err := provider.Down(ctx)
	if errors.Is(err, goose.ErrNoNextVersion) {
		fmt.Println("no migrations to roll back")
		return nil
	}
	if err != nil {
		return fmt.Errorf("roll back migration: %w", err)
	}
	fmt.Printf("rolled back %d %s (%s)\n",
		result.Source.Version,
		filepath.Base(result.Source.Path),
		result.Duration.Round(time.Millisecond),
	)
	return nil
}

// migrateStatus prints every migration and whether it has been applied.
func migrateStatus(ctx context.Context, provider *goose.Provider) error {
	statuses, err := provider.Status(ctx)
	if err != nil {
		return fmt.Errorf("read migration status: %w", err)
	}
	fmt.Printf("%-8s %-32s %-8s %s\n", "VERSION", "NAME", "STATE", "APPLIED AT")
	for _, status := range statuses {
		appliedAt := "-"
		if !status.AppliedAt.IsZero() {
			appliedAt = status.AppliedAt.Format(time.RFC3339)
		}
		fmt.Printf("%-8d %-32s %-8s %s\n",
			status.Source.Version,
			filepath.Base(status.Source.Path),
			status.State,
			appliedAt,
		)
	}
	return nil
}
