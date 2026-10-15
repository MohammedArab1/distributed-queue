// Package logging builds the application's structured logger.
package logging

import (
	"log/slog"
	"os"
)

// New returns a logger that writes key/value text logs to stdout at the given
// level. Docker collects stdout, so no log file handling is needed.
func New(level slog.Level) *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}
