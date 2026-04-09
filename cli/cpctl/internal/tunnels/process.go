package tunnels

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// ManagedProcess represents a managed process with its metadata
type ManagedProcess struct {
	PID       int
	Name      string
	Cmd       string
	Args      []string
	StartTime time.Time
	Status    string // running, stopped, killed, error
	pidFile   string
	cmd       *exec.Cmd
	cancel    context.CancelFunc
	done      chan error
	mu        sync.RWMutex
}

// ProcessManager handles tunnel process lifecycle
type ProcessManager struct {
	processes map[string]*ManagedProcess
	mu        sync.RWMutex
	dataDir   string
}

// NewProcessManager creates a new process manager
func NewProcessManager(dataDir string) *ProcessManager {
	return &ProcessManager{
		processes: make(map[string]*ManagedProcess),
		dataDir:   dataDir,
	}
}

// Spawn creates and starts a new process, returning its PID
func (pm *ProcessManager) Spawn(ctx context.Context, name string, command string, args ...string) (int, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check if process already exists
	if _, exists := pm.processes[name]; exists {
		return 0, fmt.Errorf("process '%s' already exists", name)
	}

	// Create PID file path
	pidFile := fmt.Sprintf("%s/%s.pid", pm.dataDir, name)
	
	// Ensure data directory exists
	if err := os.MkdirAll(pm.dataDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(ctx)

	// Create command
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = getSysProcAttr()

	slog.Debug("spawning process", "name", name, "command", command, "args", args)

	// Start process
	if err := cmd.Start(); err != nil {
		cancel()
		return 0, fmt.Errorf("failed to start process: %w", err)
	}

	pid := cmd.Process.Pid
	done := make(chan error, 1)

	// Create managed process
	proc := &ManagedProcess{
		PID:       pid,
		Name:      name,
		Cmd:       command,
		Args:      args,
		StartTime: time.Now(),
		Status:    "running",
		pidFile:   pidFile,
		cmd:       cmd,
		cancel:    cancel,
		done:      done,
	}

	// Save PID to file
	if err := os.WriteFile(pidFile, []byte(fmt.Sprint(pid)), 0644); err != nil {
		cancel()
		_ = cmd.Process.Kill()
		return 0, fmt.Errorf("failed to save PID: %w", err)
	}

	// Store process
	pm.processes[name] = proc

	// Wait for process completion in background
	go func() {
		err := cmd.Wait()
		proc.mu.Lock()
		if err != nil {
			proc.Status = "error"
			slog.Error("process exited with error", "name", name, "pid", pid, "error", err)
		} else {
			proc.Status = "stopped"
			slog.Info("process exited normally", "name", name, "pid", pid)
		}
		proc.mu.Unlock()
		done <- err
	}()

	slog.Info("process spawned", "name", name, "pid", pid)
	return pid, nil
}

// Track starts tracking an existing process by PID
func (pm *ProcessManager) Track(name string, pid int) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check if process already exists
	if _, exists := pm.processes[name]; exists {
		return fmt.Errorf("process '%s' already tracked", name)
	}

	// Create PID file path
	pidFile := fmt.Sprintf("%s/%s.pid", pm.dataDir, name)
	
	// Ensure data directory exists
	if err := os.MkdirAll(pm.dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Save PID to file
	if err := os.WriteFile(pidFile, []byte(fmt.Sprint(pid)), 0644); err != nil {
		return fmt.Errorf("failed to save PID: %w", err)
	}

	// Create managed process for tracking
	proc := &ManagedProcess{
		PID:       pid,
		Name:      name,
		Status:    "tracked",
		pidFile:   pidFile,
		StartTime: time.Now(),
	}

	// Store process
	pm.processes[name] = proc

	slog.Info("process tracked", "name", name, "pid", pid)
	return nil
}

// Kill stops a process gracefully with SIGTERM, then SIGKILL if needed
func (pm *ProcessManager) Kill(name string) error {
	pm.mu.Lock()
	proc, exists := pm.processes[name]
	pm.mu.Unlock()

	if !exists {
		return fmt.Errorf("process '%s' not found", name)
	}

	proc.mu.Lock()
	defer proc.mu.Unlock()

	if proc.Status == "stopped" || proc.Status == "killed" {
		return nil // Already stopped
	}

	slog.Debug("killing process", "name", name, "pid", proc.PID)

	// Try graceful shutdown first
	if proc.cancel != nil {
		proc.cancel()
	}

	if proc.cmd != nil && proc.cmd.Process != nil {
		// Send SIGTERM
		if err := proc.cmd.Process.Signal(syscall.SIGTERM); err != nil {
			slog.Warn("failed to send SIGTERM", "name", name, "pid", proc.PID, "error", err)
		}

		// Wait for graceful shutdown
		select {
		case <-time.After(5 * time.Second):
			// Force kill with SIGKILL
			slog.Warn("process not responding to SIGTERM, forcing kill", "name", name, "pid", proc.PID)
			if err := proc.cmd.Process.Kill(); err != nil {
				proc.Status = "error"
				return fmt.Errorf("failed to kill process: %w", err)
			}
		case <-proc.done:
			// Process exited gracefully
		}
	} else {
		// For tracked processes without cmd, use os.FindProcess
		process, err := os.FindProcess(proc.PID)
		if err != nil {
			proc.Status = "error"
			return fmt.Errorf("failed to find process: %w", err)
		}

		// Send SIGTERM
		if err := process.Signal(syscall.SIGTERM); err != nil {
			slog.Warn("failed to send SIGTERM to tracked process", "name", name, "pid", proc.PID, "error", err)
		}

		// Wait a bit
		time.Sleep(2 * time.Second)

		// Check if still alive and force kill
		if pm.Status(name) == "running" {
			if err := process.Kill(); err != nil {
				proc.Status = "error"
				return fmt.Errorf("failed to kill tracked process: %w", err)
			}
		}
	}

	proc.Status = "killed"
	
	// Clean up PID file
	_ = os.Remove(proc.pidFile)

	// Remove from processes map
	pm.mu.Lock()
	delete(pm.processes, name)
	pm.mu.Unlock()

	slog.Info("process killed", "name", name, "pid", proc.PID)
	return nil
}

// Status returns the status of a process
func (pm *ProcessManager) Status(name string) string {
	pm.mu.RLock()
	proc, exists := pm.processes[name]
	pm.mu.RUnlock()

	if !exists {
		return "not found"
	}

	proc.mu.RLock()
	defer proc.mu.RUnlock()

	// Check if process is still alive
	if proc.cmd != nil && proc.cmd.Process != nil {
		// Check if process has exited
		if proc.cmd.ProcessState != nil {
			return "stopped"
		}
		
		// Try to send signal 0 to check if process exists
		if err := proc.cmd.Process.Signal(syscall.Signal(0)); err != nil {
			return "stopped"
		}
		return proc.Status
	}

	// For tracked processes, check if PID exists
	if proc.PID > 0 {
		process, err := os.FindProcess(proc.PID)
		if err != nil {
			return "stopped"
		}
		
		// Try to send signal 0
		if err := process.Signal(syscall.Signal(0)); err != nil {
			return "stopped"
		}
		return proc.Status
	}

	return proc.Status
}

// List returns all managed processes
func (pm *ProcessManager) List() []ManagedProcess {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	processes := make([]ManagedProcess, 0, len(pm.processes))
	for _, proc := range pm.processes {
		proc.mu.RLock()
		processes = append(processes, *proc)
		proc.mu.RUnlock()
	}

	return processes
}

// GetProcess returns a managed process by name
func (pm *ProcessManager) GetProcess(name string) (*ManagedProcess, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	proc, exists := pm.processes[name]
	if !exists {
		return nil, false
	}
	return proc, true
}
