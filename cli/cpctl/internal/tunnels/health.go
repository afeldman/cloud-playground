package tunnels

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"
)

// HealthChecker performs connectivity checks for tunnels
type HealthChecker struct{}

// NewHealthChecker creates a new health checker
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{}
}

// Check performs a TCP connectivity check to the given address
func (hc *HealthChecker) Check(addr string, maxRetries int, retryDelay time.Duration) (bool, time.Duration, error) {
	var lastErr error
	start := time.Now()

	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			time.Sleep(retryDelay)
			slog.Debug("retrying health check", "addr", addr, "attempt", i+1, "max", maxRetries)
		}

		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err == nil {
			latency := time.Since(start)
			conn.Close()
			slog.Debug("health check passed", "addr", addr, "latency", latency)
			return true, latency, nil
		}
		lastErr = err
	}

	totalTime := time.Since(start)
	slog.Debug("health check failed", "addr", addr, "error", lastErr, "total_time", totalTime)
	return false, totalTime, lastErr
}

// CheckWithContext performs a TCP connectivity check with context cancellation
func (hc *HealthChecker) CheckWithContext(ctx context.Context, addr string, maxRetries int, retryDelay time.Duration) (bool, time.Duration, error) {
	var lastErr error
	start := time.Now()

	for i := 0; i < maxRetries; i++ {
		select {
		case <-ctx.Done():
			return false, time.Since(start), ctx.Err()
		default:
			// Continue with check
		}

		if i > 0 {
			select {
			case <-ctx.Done():
				return false, time.Since(start), ctx.Err()
			case <-time.After(retryDelay):
				// Wait for retry delay
			}
			slog.Debug("retrying health check", "addr", addr, "attempt", i+1, "max", maxRetries)
		}

		// Use a shorter timeout for each attempt when we have a context
		dialCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		var d net.Dialer
		conn, err := d.DialContext(dialCtx, "tcp", addr)
		if err == nil {
			latency := time.Since(start)
			conn.Close()
			slog.Debug("health check passed", "addr", addr, "latency", latency)
			return true, latency, nil
		}
		lastErr = err
	}

	totalTime := time.Since(start)
	slog.Debug("health check failed", "addr", addr, "error", lastErr, "total_time", totalTime)
	return false, totalTime, lastErr
}

// CheckTunnel performs a health check for a tunnel by name
func (hc *HealthChecker) CheckTunnel(tunnel *TunnelInfo) (bool, string) {
	addr := fmt.Sprintf("localhost:%d", tunnel.LocalPort)
	connected, latency, err := hc.Check(addr, 3, 1*time.Second)

	if connected {
		return true, fmt.Sprintf("✅ Connected (latency: %v)", latency)
	}

	if err != nil {
		return false, fmt.Sprintf("❌ Failed: %v", err)
	}

	return false, "❌ Not connected"
}

// CheckAllTunnels performs health checks for all provided tunnels
func (hc *HealthChecker) CheckAllTunnels(tunnels []TunnelInfo) map[string]string {
	results := make(map[string]string)
	
	for _, tunnel := range tunnels {
		connected, status := hc.CheckTunnel(&tunnel)
		results[tunnel.Name] = status
		slog.Debug("tunnel health check", "name", tunnel.Name, "connected", connected, "status", status)
	}
	
	return results
}