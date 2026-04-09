package tunnels

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"time"

	"cpctl/internal/config"
)

// KubernetesTunnel manages kubectl port-forward tunnels
type KubernetesTunnel struct {
	name      string
	namespace string
	service   string
	localPort int
	remotePort int
	kubeconfig string
	pm        *ProcessManager
	pidFile   string
}

// NewKubernetesTunnel creates a new Kubernetes tunnel
func NewKubernetesTunnel(cfg *config.Config, name string) (*KubernetesTunnel, error) {
	tunnelCfg, ok := cfg.Tunnels[name]
	if !ok {
		return nil, fmt.Errorf("tunnel not found: %s", name)
	}

	if tunnelCfg.Type != "kubernetes" {
		return nil, fmt.Errorf("tunnel is not kubernetes type: %s", tunnelCfg.Type)
	}

	kubeconfig := os.ExpandEnv(cfg.Kind.Kubeconfig)

	return &KubernetesTunnel{
		name:       name,
		namespace:  tunnelCfg.Namespace,
		service:    tunnelCfg.Service,
		localPort:  tunnelCfg.LocalPort,
		remotePort: tunnelCfg.RemotePort,
		kubeconfig: kubeconfig,
	}, nil
}

// Start begins the port-forward tunnel
func (kt *KubernetesTunnel) Start(ctx context.Context) error {
	dataDir := fmt.Sprintf("%s/tunnels", config.Cfg.Playground.DataDir)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}

	pm := NewProcessManager(dataDir)
	kt.pm = pm

	// Construct kubectl port-forward command
	target := fmt.Sprintf("svc/%s", kt.service)
	portForward := fmt.Sprintf("%d:%d", kt.localPort, kt.remotePort)

	slog.Debug(
		"starting kubernetes tunnel",
		"name", kt.name,
		"namespace", kt.namespace,
		"service", kt.service,
		"port_forward", portForward,
	)

	// Start port-forward using Spawn
	pid, err := pm.Spawn(
		ctx,
		kt.name,
		"kubectl",
		"--kubeconfig", kt.kubeconfig,
		"-n", kt.namespace,
		"port-forward",
		target,
		portForward,
	)
	if err != nil {
		return err
	}

	// Save tunnel metadata using TunnelManager
	tm := New(config.Cfg.Playground.DataDir)
	tunnelInfo := TunnelInfo{
		Name:       kt.name,
		PID:        pid,
		Type:       "kubernetes",
		LocalPort:  kt.localPort,
		RemoteHost: fmt.Sprintf("%s.%s.svc.cluster.local", kt.service, kt.namespace),
		RemotePort: kt.remotePort,
	}
	
	if err := tm.Save(tunnelInfo); err != nil {
		slog.Error("failed to save tunnel metadata", "name", kt.name, "error", err)
		// Don't fail the tunnel start if metadata save fails, but log it
	}

	// Wait for tunnel to be ready (TCP connectivity check)
	if err := kt.waitForReady(ctx, 10*time.Second); err != nil {
		_ = pm.Kill(kt.name)
		_ = tm.Delete(kt.name) // Clean up metadata
		return err
	}

	slog.Info("kubernetes tunnel started", "name", kt.name, "pid", pid)
	return nil
}

// Stop terminates the port-forward tunnel
func (kt *KubernetesTunnel) Stop() error {
	if kt.pm == nil {
		return fmt.Errorf("tunnel not running")
	}

	if err := kt.pm.Kill(kt.name); err != nil {
		slog.Error("failed to stop tunnel", "name", kt.name, "error", err)
		return err
	}

	// Clean up tunnel metadata
	tm := New(config.Cfg.Playground.DataDir)
	if err := tm.Delete(kt.name); err != nil {
		slog.Warn("failed to delete tunnel metadata", "name", kt.name, "error", err)
		// Don't fail the stop if metadata delete fails
	}

	slog.Info("kubernetes tunnel stopped", "name", kt.name)
	return nil
}

// Status returns tunnel status
func (kt *KubernetesTunnel) Status() (TunnelStatus, error) {
	status := TunnelStatus{
		Name:        kt.name,
		Type:        "kubernetes",
		LocalPort:   kt.localPort,
		RemotePort:  kt.remotePort,
		RemoteHost:  fmt.Sprintf("%s.%s.svc.cluster.local", kt.service, kt.namespace),
	}

	if kt.pm == nil {
		status.State = "stopped"
		return status, nil
	}

	// Get process status
	processStatus := kt.pm.Status(kt.name)
	
	// Try to get PID from process manager
	if proc, exists := kt.pm.GetProcess(kt.name); exists {
		status.PID = proc.PID
	}

	if processStatus == "running" || processStatus == "tracked" {
		status.State = "running"
		// Try to connect to verify
		if err := kt.checkConnectivity(); err != nil {
			status.State = "unhealthy"
			status.Error = err.Error()
		}
	} else {
		status.State = processStatus
	}

	return status, nil
}

// waitForReady waits for the tunnel to be ready for connections
func (kt *KubernetesTunnel) waitForReady(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("tunnel readiness timeout: %w", ctx.Err())
		case <-ticker.C:
			if err := kt.checkConnectivity(); err == nil {
				return nil
			}
		}
	}
}

// checkConnectivity verifies tunnel is accepting connections
func (kt *KubernetesTunnel) checkConnectivity() error {
	addr := fmt.Sprintf("localhost:%d", kt.localPort)
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}
