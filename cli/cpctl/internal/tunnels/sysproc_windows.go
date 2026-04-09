// +build windows

package tunnels

import "syscall"

// getSysProcAttr returns platform-specific process attributes
func getSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}
