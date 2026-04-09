package logger

import (
	"context"
	"log/slog"
)

type discardHandler struct{}

func (d discardHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (d discardHandler) Handle(context.Context, slog.Record) error {
	return nil
}

func (d discardHandler) WithAttrs([]slog.Attr) slog.Handler {
	return d
}

func (d discardHandler) WithGroup(string) slog.Handler {
	return d
}
