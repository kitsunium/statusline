//go:build windows

package daemonlink

import "syscall"

// detached hides the daemon's console window.
func detached() *syscall.SysProcAttr { return &syscall.SysProcAttr{HideWindow: true} }
