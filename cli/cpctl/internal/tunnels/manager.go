package tunnels

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// TunnelManager holds information about active tunnels
type TunnelManager struct {
	DataDir string
}

// TunnelInfo represents a single active tunnel
type TunnelInfo struct {
	Name       string
	PID        int
	Type       string
	LocalPort  int
	RemoteHost string
	RemotePort int
}

// New creates a new TunnelManager
func New(dataDir string) *TunnelManager {
	return &TunnelManager{
		DataDir: filepath.Join(dataDir, "tunnels"),
	}
}

// Save stores tunnel PID and metadata
func (tm *TunnelManager) Save(info TunnelInfo) error {
	if err := os.MkdirAll(tm.DataDir, 0755); err != nil {
		return err
	}

	pidFile := filepath.Join(tm.DataDir, fmt.Sprintf("%s.pid", info.Name))
	metaFile := filepath.Join(tm.DataDir, fmt.Sprintf("%s.json", info.Name))

	// Write PID
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(info.PID)), 0644); err != nil {
		return err
	}

	// Write metadata (simplified as text for now)
	meta := fmt.Sprintf("type=%s\nlocal_port=%d\nremote_host=%s\nremote_port=%d\n",
		info.Type, info.LocalPort, info.RemoteHost, info.RemotePort)
	if err := os.WriteFile(metaFile, []byte(meta), 0644); err != nil {
		return err
	}

	slog.Info("tunnel saved", "name", info.Name, "pid", info.PID)
	return nil
}

// Load retrieves tunnel information by name
func (tm *TunnelManager) Load(name string) (*TunnelInfo, error) {
	pidFile := filepath.Join(tm.DataDir, fmt.Sprintf("%s.pid", name))
	metaFile := filepath.Join(tm.DataDir, fmt.Sprintf("%s.json", name))

	pidBytes, err := os.ReadFile(pidFile)
	if err != nil {
		return nil, fmt.Errorf("tunnel not found: %s", name)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(pidBytes)))
	if err != nil {
		return nil, err
	}

	metaBytes, err := os.ReadFile(metaFile)
	if err != nil {
		return nil, err
	}

	// Parse metadata
	info := &TunnelInfo{
		Name: name,
		PID:  pid,
	}

	for _, line := range strings.Split(string(metaBytes), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, val := parts[0], parts[1]
		switch key {
		case "type":
			info.Type = val
		case "local_port":
			info.LocalPort, _ = strconv.Atoi(val)
		case "remote_host":
			info.RemoteHost = val
		case "remote_port":
			info.RemotePort, _ = strconv.Atoi(val)
		}
	}

	return info, nil
}

// List returns all active tunnels
func (tm *TunnelManager) List() ([]TunnelInfo, error) {
	info, err := os.ReadDir(tm.DataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []TunnelInfo{}, nil
		}
		return nil, err
	}

	var tunnels []TunnelInfo
	seen := make(map[string]bool)

	for _, entry := range info {
		if entry.IsDir() {
			continue
		}

		// Only process .pid files
		if !strings.HasSuffix(entry.Name(), ".pid") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".pid")
		if seen[name] {
			continue
		}
		seen[name] = true

		tunnel, err := tm.Load(name)
		if err == nil {
			tunnels = append(tunnels, *tunnel)
		}
	}

	return tunnels, nil
}

// Delete removes tunnel files
func (tm *TunnelManager) Delete(name string) error {
	pidFile := filepath.Join(tm.DataDir, fmt.Sprintf("%s.pid", name))
	metaFile := filepath.Join(tm.DataDir, fmt.Sprintf("%s.json", name))

	if err := os.Remove(pidFile); err != nil && !os.IsNotExist(err) {
		return err
	}

	if err := os.Remove(metaFile); err != nil && !os.IsNotExist(err) {
		return err
	}

	slog.Info("tunnel deleted", "name", name)
	return nil
}
