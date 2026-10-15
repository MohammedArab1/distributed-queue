// Command server runs the jobqueue server. It loads configuration, connects to
// Postgres, and shuts down cleanly when it receives SIGINT or SIGTERM.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/MohammedArab1/jobqueue/internal/config"
	"github.com/MohammedArab1/jobqueue/internal/logging"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// run holds all the startup work. Keeping it separate from main means its
// deferred calls always run before the process exits.
func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.LoadServer()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	logger := logging.New(cfg.LogLevel)

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("create pg pool: %w", err)
	}
	defer pool.Close()

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}
	logger.Info("connected to postgres")

	<-ctx.Done() // wait for Ctrl-C
	logger.Info("shutting down")
	return nil
}
