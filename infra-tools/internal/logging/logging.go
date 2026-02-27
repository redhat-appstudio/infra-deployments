// Package logging provides shared slog configuration for infra-tools CLIs.
// It supports dual-output logging (stderr for user-facing messages, file for
// debug-level details) using charmbracelet/log as the handler backend.
package logging

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	charmlog "github.com/charmbracelet/log"
)

// Setup configures the global slog logger. If logFile is non-empty, a debug-level
// file handler is added alongside the stderr INFO handler. Returns a cleanup
// function (may be nil) and any error.
func Setup(logFile string) (func(), error) {
	stderrHandler := charmlog.NewWithOptions(os.Stderr, charmlog.Options{
		Level: charmlog.InfoLevel,
	})

	if logFile == "" {
		slog.SetDefault(slog.New(stderrHandler))
		return nil, nil
	}

	f, err := os.Create(logFile)
	if err != nil {
		return nil, fmt.Errorf("opening log file %s: %w", logFile, err)
	}

	fileHandler := charmlog.NewWithOptions(f, charmlog.Options{
		Level:           charmlog.DebugLevel,
		ReportTimestamp: true,
	})

	multi := &multiHandler{handlers: []slog.Handler{stderrHandler, fileHandler}}
	slog.SetDefault(slog.New(multi))

	return func() { _ = f.Close() }, nil
}

// Fatal logs an error message and exits the process.
func Fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}

// multiHandler fans out log records to multiple slog.Handler instances.
type multiHandler struct {
	handlers []slog.Handler
}

func (m *multiHandler) Enabled(_ context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(context.Background(), level) {
			return true
		}
	}
	return false
}

func (m *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, h := range m.handlers {
		if h.Enabled(ctx, r.Level) {
			if err := h.Handle(ctx, r); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithAttrs(attrs)
	}
	return &multiHandler{handlers: handlers}
}

func (m *multiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithGroup(name)
	}
	return &multiHandler{handlers: handlers}
}
