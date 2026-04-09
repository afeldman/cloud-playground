package tunnels

import (
	"context"
)

// TunnelConfig represents the configuration for a tunnel from .cpctl.yaml
type TunnelConfig struct {
	Type       string
	Method     string
	Namespace  string
	Service    string
	LocalPort  int
	RemotePort int
	RemoteHost string
	SSHUser    string
	AutoStart  bool
}

// TunnelStatus represents the current state of a tunnel
type TunnelStatus struct {
	Name       string // Tunnel name
	Type       string // kubernetes or aws-ssm
	State      string // running, stopped, unhealthy
	PID        int    // Process ID
	LocalPort  int    // Local port number
	RemotePort int    // Remote port number
	RemoteHost string // Remote host (service or RDS endpoint)
	Error      string // Error message if unhealthy
}

// Tunnel represents a generic tunnel interface
type Tunnel interface {
	// Start begins the tunnel
	Start(ctx context.Context) error

	// Stop terminates the tunnel
	Stop() error

	// Status returns current tunnel status
	Status() (TunnelStatus, error)
}
