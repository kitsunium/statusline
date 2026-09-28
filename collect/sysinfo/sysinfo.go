package sysinfo

import (
	"os"
	"runtime"
	"strings"

	"github.com/kitsunium/statusline/snapshot"
)

const (
	// dockerEnvPath is the file Docker creates at the container's root.
	dockerEnvPath string = "/.dockerenv"
	// cgroupPath names the init process's control groups.
	cgroupPath string = "/proc/1/cgroup"
	// dockerIdentifier marks a Docker control group.
	dockerIdentifier string = "docker"
)

func detectOS() int {
	switch runtime.GOOS {
	case "linux":
		return snapshot.OSLinux
	case "darwin":
		return snapshot.OSDarwin
	case "windows":
		return snapshot.OSWindows
	default:
		return snapshot.OSUnknown
	}
}

func isDocker() bool {
	if _, err := os.Stat(dockerEnvPath); err == nil {
		return true
	}
	data, err := os.ReadFile(cgroupPath)
	return err == nil && strings.Contains(string(data), dockerIdentifier)
}
