//go:build !windows

package daemonlink

import "syscall"

// detached gives the daemon its own session, away from the host's terminal.
func detached() *syscall.SysProcAttr { return &syscall.SysProcAttr{Setsid: true} }
