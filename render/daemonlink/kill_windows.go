//go:build windows

package daemonlink

import "os"

// kill ends a stuck daemon.
func kill(pid int) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return p.Kill()
}
