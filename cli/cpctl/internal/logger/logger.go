package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Level string
type Format string

const (
	Quiet Level = "quiet"
	Info  Level = "info"
	Debug Level = "debug"

	Text Format = "text"
	JSON Format = "json"
	TOON Format = "toon"

	// Rolling logger defaults
	defaultLogDir       = "./data/logs"
	defaultLogFile      = "cpctl.log"
	defaultMaxSizeBytes = 10 * 1024 * 1024 // 10MB
	defaultMaxBackups   = 5
	defaultMaxAgeDays   = 7
)

func New(level Level, format Format) *slog.Logger {
	// Ensure log directory exists
	logDir := defaultLogDir
	if err := os.MkdirAll(logDir, 0755); err != nil {
		// Fallback: write to stderr only
		return fallbackTextLogger(level)
	}

	logPath := filepath.Join(logDir, defaultLogFile)

	// Configure lumberjack rolling writer
	lumberWriter := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    10, // MB
		MaxBackups: defaultMaxBackups,
		MaxAge:     defaultMaxAgeDays,
		Compress:   true,
	}

	// Convert level string to zap level
	zapLevel := zapLevelFromString(level)

	// Configure zap encoder
	encoderCfg := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
	}

	var encoder zapcore.Encoder
	switch format {
	case JSON:
		encoder = zapcore.NewJSONEncoder(encoderCfg)
	default:
		// TEXT and TOON both use console encoder
		encoder = zapcore.NewConsoleEncoder(encoderCfg)
	}

	// Create output writer (file + stderr)
	writer := io.MultiWriter(lumberWriter, os.Stderr)

	// Create zap core
	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(writer),
		zapLevel,
	)

	// Create zap logger with caller info
	zapLogger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	// Wrap it for slog compatibility
	return newWrappedSlogLogger(zapLogger)
}

// ZapSlogAdapter wraps zap.Logger to be compatible with slog handlers
// This bridges between zap's internal logging and slog's public API
type ZapSlogAdapter struct {
	zapLogger *zap.Logger
}

// Handle implements slog.Handler interface for zap
func (a *ZapSlogAdapter) Handle(ctx context.Context, r slog.Record) error {
	// Convert slog Record to zap fields
	fields := []zap.Field{
		zap.String("msg", r.Message),
	}

	// Add attributes as zap fields
	r.Attrs(func(attr slog.Attr) bool {
		fields = append(fields, zap.Any(attr.Key, attr.Value.Any()))
		return true
	})

	// Log based on level
	switch r.Level {
	case slog.LevelDebug:
		a.zapLogger.Debug(r.Message, fields...)
	case slog.LevelInfo:
		a.zapLogger.Info(r.Message, fields...)
	case slog.LevelWarn:
		a.zapLogger.Warn(r.Message, fields...)
	default: // Error and higher
		a.zapLogger.Error(r.Message, fields...)
	}

	return nil
}

func (a *ZapSlogAdapter) WithAttrs(attrs []slog.Attr) slog.Handler {
	return a // For simplicity, just return self
}

func (a *ZapSlogAdapter) WithGroup(name string) slog.Handler {
	return a
}

func (a *ZapSlogAdapter) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func newWrappedSlogLogger(zapLog *zap.Logger) *slog.Logger {
	adapter := &ZapSlogAdapter{zapLogger: zapLog}
	return slog.New(adapter)
}

// fallbackTextLogger returns a simple text logger to stderr (if file logging fails)
func fallbackTextLogger(level Level) *slog.Logger {
	slogLevel := slog.LevelWarn
	switch level {
	case Debug:
		slogLevel = slog.LevelDebug
	case Info:
		slogLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: slogLevel}
	handler := slog.NewTextHandler(os.Stderr, opts)
	return slog.New(handler)
}

// zapLevelFromString converts Level to zapcore.Level
func zapLevelFromString(level Level) zapcore.Level {
	switch level {
	case Debug:
		return zapcore.DebugLevel
	case Info:
		return zapcore.InfoLevel
	case Quiet:
		return zapcore.WarnLevel
	default:
		return zapcore.InfoLevel
	}
}
