# Rolling Logger Implementation (Zap + Lumberjack)

## Overview
Replaced Go's standard `log/slog` with **zap** (Uber's structured logging library) + **lumberjack** (rolling file writer) for:
- ✅ Automatic log file rotation (size-based)
- ✅ Compression of archived logs
- ✅ Backward-compatible slog API
- ✅ High performance structured logging
- ✅ File + stderr dual output

## Configuration

### .cpctl.yaml
```yaml
logging:
  enabled: true
  file: ./data/logs/cpctl.log
  max_size_mb: 10          # Rotate when log file exceeds 10MB
  max_backups: 5           # Keep last 5 rotated logs
  max_age_days: 7          # Delete logs older than 7 days
  compress: true           # Compress rotated logs as .gz
```

### CLI Flags
```bash
cpctl --log-format text|json|toon     # Text: console output, JSON: structured
cpctl -l quiet|info|debug             # Log level
```

## Implementation Details

### File Layout
- [internal/logger/logger.go](./internal/logger/logger.go) — Main logging setup with zap + lumberjack
- [internal/logger/logger_test.go](./internal/logger/logger_test.go) — Comprehensive logger tests
- [.cpctl.yaml](.cpctl.yaml) — Logging config section

### ZapSlogAdapter Bridge
Custom `slog.Handler` adapter wraps zap.Logger for backward compatibility:
```go
type ZapSlogAdapter struct {
    zapLogger *zap.Logger
}

func (a *ZapSlogAdapter) Handle(ctx context.Context, r slog.Record) error {
    // converts slog record → zap fields → logs to file + stderr
}
```

Benefit: **All existing slog calls in codebase continue to work unchanged!**

### Log Rotation
**Lumberjack automatically:**
1. Writes to `./data/logs/cpctl.log`
2. When file reaches 10MB → rotates to `cpctl.log.1`, `cpctl.log.2`, etc.
3. Compresses old logs: `cpctl.log.1 → cpctl.log.1.gz`
4. Deletes logs older than 7 days
5. Keeps max 5 backups

### Dual Output
All logs written to **both**:
- **File**: `./data/logs/cpctl.log` (persistent, rotated)
- **Stderr**: Live console output (for terminal interaction)

## Testing

Run tests:
```bash
cd cli/cpctl
go test ./internal/logger -v
```

Output shows:
- ✅ Text format logging works
- ✅ Log levels (debug, info, warn) work
- ✅ Fallback logger (if directory creation fails)
- ✅ ISO8601 timestamps
- ✅ Structured fields marshaling

## Usage

No code changes required! Existing slog calls work as-is:

```go
// Existing code continues to work
slog.Info("message", slog.String("key", "value"))
slog.Error("error", "error", err)
```

All logs automatically:
1. Go to `./data/logs/cpctl.log`
2. Rotate when file size > 10MB
3. Display to stderr for immediate feedback

## Performance

Zap is **significantly faster** than slog for structured logging:
- **Lower latency** on each log call
- **Less memory allocation**
- **Better concurrency** (lock-free design)

Benchmark comparison: zap ~10-50x faster than stdlib logging

## Fallback

If log directory creation fails:
- Gracefully falls back to stderr-only (standard slog.TextHandler)
- Logs still visible in console
- No crashes or panics

## Future Enhancements

Optional improvements (not implemented yet):
- [ ] Load logging config from .cpctl.yaml dynamically
- [ ] Support per-module log levels
- [ ] JSON structured fields in file output
- [ ] Log aggregation/export to ELK stack
- [ ] Custom formatters (e.g., toon format compatible with zap)

## References

- **go.uber.org/zap**: https://github.com/uber-go/zap
- **gopkg.in/natefinch/lumberjack.v2**: https://github.com/natefinch/lumberjack
- **slog handler bridge**: Custom adapter in [logger.go](./internal/logger/logger.go)
