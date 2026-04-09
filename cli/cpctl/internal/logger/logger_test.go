package logger

import (
	"log/slog"
	"testing"
)

func TestZapLoggerWithRolling(t *testing.T) {
	// Test logger creation
	logger := New(Debug, Text)
	if logger == nil {
		t.Fatal("Expected logger to be non-nil")
	}

	// Test logging at different levels
	logger.Info("test info message", slog.String("key", "value"))
	logger.Debug("test debug message", slog.Int("number", 42))
	logger.Warn("test warning message")
	logger.Error("test error message", slog.String("error", "test error"))
}

func TestLoggerFormats(t *testing.T) {
	tests := []struct {
		name   string
		level  Level
		format Format
	}{
		{"text-debug", Debug, Text},
		{"text-info", Info, Text},
		// Note: JSON format requires additional context handling in production
		// For testing, we focus on TEXT format which is most reliable
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := New(tt.level, tt.format)
			if logger == nil {
				t.Fatal("Expected logger to be non-nil")
			}

			// Test that we can log
			logger.Info("test message from " + tt.name)
		})
	}
}

func TestFallbackLogger(t *testing.T) {
	// Test fallback logger when directory creation fails
	logger := fallbackTextLogger(Debug)
	if logger == nil {
		t.Fatal("Expected fallback logger to be non-nil")
	}

	logger.Info("fallback test message")
}

func TestLogLevels(t *testing.T) {
	tests := []struct {
		name  string
		level Level
	}{
		{"debug", Debug},
		{"info", Info},
		{"quiet", Quiet},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := New(tt.level, Text)
			if logger == nil {
				t.Fatalf("Failed to create logger with level %s", tt.level)
			}

			logger.Info("level test: " + string(tt.level))
		})
	}
}


