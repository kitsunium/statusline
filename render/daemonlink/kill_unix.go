//go:build !windows

package daemonlink

import "syscall"

// kill ends a stuck daemon; the kernel refuses another user's process.
func kill(pid int) error { return syscall.Kill(pid, syscall.SIGKILL) }
