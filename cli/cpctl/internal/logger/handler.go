package logger

import (
	"context"
	"log/slog"
)

// TOONHandler decorates a slog.Handler with human-friendly console output.
// It MUST NOT affect structured logging or CI output.
type TOONHandler struct {
	next slog.Handler
}

func NewTOONHandler(next slog.Handler) slog.Handler {
	return &TOONHandler{next: next}
}

func (h *TOONHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *TOONHandler) Handle(ctx context.Context, r slog.Record) error {
	// UI output: message only, no attrs, no secrets
	switch r.Level {
	case slog.LevelError:
		Failure(r.Message)
	case slog.LevelWarn:
		Warning(r.Message)
	case slog.LevelInfo:
		Message(r.Message)
	default:
		// Debug intentionally ignored in UI
	}

	return h.next.Handle(ctx, r)
}

func (h *TOONHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &TOONHandler{
		next: h.next.WithAttrs(attrs),
	}
}

func (h *TOONHandler) WithGroup(name string) slog.Handler {
	return &TOONHandler{
		next: h.next.WithGroup(name),
	}
}
