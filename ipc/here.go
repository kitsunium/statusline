package ipc

import (
	"os"
	"path/filepath"
)

// here resolves symbolic links so that a client started through a link and
// the daemon it starts agree on the executable, hence on the instance.
func here(getenv func(string) string) (Instance, error) {
	exe, err := os.Executable()
	if err != nil {
		return Instance{}, err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	configDir := getenv("CLAUDE_CONFIG_DIR")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return Instance{}, err
		}
		configDir = filepath.Join(home, ".claude")
	}
	runtimeDir := getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = os.TempDir()
	}
	return locate(LocateInput{RuntimeDir: runtimeDir, UID: os.Getuid(), ConfigDir: configDir, Executable: exe}), nil
}
