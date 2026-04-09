package tunnels

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"time"

	"cpctl/internal/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// SSMTunnel manages AWS SSM Session Manager tunnels
type SSMTunnel struct {
	name           string
	instanceID     string
	localPort      int
	remoteHost     string
	remotePort     int
	sshUser        string
	awsConfig      aws.Config
	pm             *ProcessManager
	pidFile        string
}

// NewSSMTunnel creates a new SSM tunnel
func NewSSMTunnel(cfg *config.Config, name string) (*SSMTunnel, error) {
	tunnelCfg, ok := cfg.Tunnels[name]
	if !ok {
		return nil, fmt.Errorf("tunnel not found: %s", name)
	}

	if tunnelCfg.Type != "aws-ssm" {
		return nil, fmt.Errorf("tunnel is not aws-ssm type: %s", tunnelCfg.Type)
	}

	// Load AWS config (uses default credential chain)
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.AWS.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &SSMTunnel{
		name:       name,
		instanceID: tunnelCfg.RemoteHost, // Actually the bastion/instance ID
		localPort:  tunnelCfg.LocalPort,
		remoteHost: tunnelCfg.RemoteHost,
		remotePort: tunnelCfg.RemotePort,
		sshUser:    tunnelCfg.SSHUser,
		awsConfig:  awsCfg,
	}, nil
}

// Start begins the SSM tunnel
func (st *SSMTunnel) Start(ctx context.Context) error {
	dataDir := fmt.Sprintf("%s/tunnels", config.Cfg.Playground.DataDir)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}

	pm := NewProcessManager(dataDir)
	st.pm = pm

	// Verify instance exists via SSM
	if err := st.verifyInstance(ctx); err != nil {
		return fmt.Errorf("failed to verify instance: %w", err)
	}

	slog.Debug(
		"starting SSM tunnel",
		"name", st.name,
		"instance_id", st.instanceID,
		"remote_host", st.remoteHost,
		"remote_port", st.remotePort,
		"local_port", st.localPort,
	)

	// Start port-forward via SSM
	// Format: aws-cli-start-session with port-forward plugin
	pid, err := pm.Spawn(
		ctx,
		st.name,
		"aws",
		"ssm", "start-session",
		"--target", st.instanceID,
		"--document-name", "AWS-StartPortForwardingSession",
		"--parameters", fmt.Sprintf("localPortNumber=%d,portNumber=%d,host=%s",
			st.localPort, st.remotePort, st.remoteHost),
	)
	if err != nil {
		return err
	}

	// Save tunnel metadata using TunnelManager
	tm := New(config.Cfg.Playground.DataDir)
	tunnelInfo := TunnelInfo{
		Name:       st.name,
		PID:        pid,
		Type:       "aws-ssm",
		LocalPort:  st.localPort,
		RemoteHost: st.remoteHost,
		RemotePort: st.remotePort,
	}
	
	if err := tm.Save(tunnelInfo); err != nil {
		slog.Error("failed to save tunnel metadata", "name", st.name, "error", err)
		// Don't fail the tunnel start if metadata save fails, but log it
	}

	// Wait for tunnel to be ready
	if err := st.waitForReady(ctx, 15*time.Second); err != nil {
		_ = pm.Kill(st.name)
		_ = tm.Delete(st.name) // Clean up metadata
		return err
	}

	slog.Info("SSM tunnel started", "name", st.name, "pid", pid)
	return nil
}

// Stop terminates the SSM tunnel
func (st *SSMTunnel) Stop() error {
	if st.pm == nil {
		return fmt.Errorf("tunnel not running")
	}

	if err := st.pm.Kill(st.name); err != nil {
		slog.Error("failed to stop tunnel", "name", st.name, "error", err)
		return err
	}

	slog.Info("SSM tunnel stopped", "name", st.name)
	return nil
}

// Status returns tunnel status
func (st *SSMTunnel) Status() (TunnelStatus, error) {
	status := TunnelStatus{
		Name:       st.name,
		Type:       "aws-ssm",
		LocalPort:  st.localPort,
		RemotePort: st.remotePort,
		RemoteHost: st.remoteHost,
	}

	if st.pm == nil {
		status.State = "stopped"
		return status, nil
	}

	// Get process status
	processStatus := st.pm.Status(st.name)
	
	// Try to get PID from process manager
	if proc, exists := st.pm.GetProcess(st.name); exists {
		status.PID = proc.PID
	}

	if processStatus == "running" || processStatus == "tracked" {
		status.State = "running"
		// Try to connect to verify
		if err := st.checkConnectivity(); err != nil {
			status.State = "unhealthy"
			status.Error = err.Error()
		}
	} else {
		status.State = processStatus
	}

	return status, nil
}

// verifyInstance checks if instance is accessible via SSM
func (st *SSMTunnel) verifyInstance(ctx context.Context) error {
	client := ssm.NewFromConfig(st.awsConfig)

	// Ping the instance
	_, err := client.DescribeInstanceInformation(ctx, &ssm.DescribeInstanceInformationInput{
		Filters: []types.InstanceInformationStringFilter{
			{
				Key:      aws.String("InstanceIds"),
				Values:   []string{st.instanceID},
			},
		},
	})

	if err != nil {
		return fmt.Errorf("instance not accessible via SSM: %w", err)
	}

	return nil
}

// waitForReady waits for the tunnel to be ready for connections
func (st *SSMTunnel) waitForReady(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("tunnel readiness timeout: %w", ctx.Err())
		case <-ticker.C:
			if err := st.checkConnectivity(); err == nil {
				return nil
			}
		}
	}
}

// checkConnectivity verifies tunnel is accepting connections
func (st *SSMTunnel) checkConnectivity() error {
	addr := fmt.Sprintf("localhost:%d", st.localPort)
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}
