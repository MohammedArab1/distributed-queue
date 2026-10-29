// Command migrate applies, rolls back or reports schema migrations. It is the
// single migration entry point for local development, tests and deployment.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/MohammedArab1/jobqueue/internal/config"
	"github.com/MohammedArab1/jobqueue/internal/store"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// run holds all the work so its deferred calls run before the process exits.
func run() error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: %s up|down|status", os.Args[0])
	}
	command := os.Args[1]

	cfg, err := config.LoadServer()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return store.Migrate(ctx, cfg.DatabaseURL, command)
}
