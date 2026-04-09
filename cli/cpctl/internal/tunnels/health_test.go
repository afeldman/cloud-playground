package tunnels

import (
	"net"
	"testing"
	"time"
)

// startTestListener starts a TCP listener on a random port and returns the address + cleanup func.
func startTestListener(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start test listener: %v", err)
	}
	t.Cleanup(func() { ln.Close() })
	// Accept connections in background so dialers don't hang
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			c.Close()
		}
	}()
	return ln.Addr().String()
}

func TestHealthChecker_Check_Connected(t *testing.T) {
	addr := startTestListener(t)
	hc := NewHealthChecker()

	ok, latency, err := hc.Check(addr, 3, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Check error: %v", err)
	}
	if !ok {
		t.Error("expected connected=true")
	}
	if latency <= 0 {
		t.Error("expected positive latency")
	}
}

func TestHealthChecker_Check_Unreachable(t *testing.T) {
	hc := NewHealthChecker()
	// Use a port that is almost certainly not open
	ok, _, err := hc.Check("127.0.0.1:19999", 1, 10*time.Millisecond)
	if ok {
		t.Error("expected connected=false for unreachable address")
	}
	if err == nil {
		t.Error("expected an error for unreachable address")
	}
}

func TestHealthChecker_CheckTunnel_Connected(t *testing.T) {
	addr := startTestListener(t)
	var port int
	_, err := net.LookupPort("tcp", "0")
	_ = err

	// Extract port from addr
	ln, _ := net.ResolveTCPAddr("tcp", addr)
	port = ln.Port

	hc := NewHealthChecker()
	tunnel := &TunnelInfo{
		Name:      "test",
		LocalPort: port,
	}

	ok, status := hc.CheckTunnel(tunnel)
	if !ok {
		t.Errorf("expected connected=true, status: %s", status)
	}
}

func TestHealthChecker_CheckAllTunnels(t *testing.T) {
	addr1 := startTestListener(t)
	ln1, _ := net.ResolveTCPAddr("tcp", addr1)

	hc := NewHealthChecker()
	tunnels := []TunnelInfo{
		{Name: "up", LocalPort: ln1.Port},
		{Name: "down", LocalPort: 19998}, // almost certainly closed
	}

	results := hc.CheckAllTunnels(tunnels)
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
	if results["up"] == "" {
		t.Error("expected status for 'up' tunnel")
	}
	if results["down"] == "" {
		t.Error("expected status for 'down' tunnel")
	}
}
