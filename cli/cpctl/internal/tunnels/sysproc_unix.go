// +build !windows

package tunnels

import "syscall"

// getSysProcAttr returns platform-specific process attributes
func getSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setsid: true, // Create new session (process group on Unix)
	}
}
