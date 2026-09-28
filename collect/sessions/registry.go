package sessions

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/kitsunium/statusline/collect/state"
)

const (
	// configDirEnv relocates the whole configuration directory.
	configDirEnv string = "CLAUDE_CONFIG_DIR"
	// busyStatus is the status the host writes while a turn is running.
	busyStatus string = "busy"
)

// sessionFile is the part of a registry entry read here.
type sessionFile struct {
	PID       int    `json:"pid"`
	SessionID string `json:"sessionId"`
	Status    string `json:"status"`
}

// registryDir is <config>/sessions; empty without a home.
func registryDir() string {
	base := os.Getenv(configDirEnv)
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		base = filepath.Join(home, ".claude")
	}
	return filepath.Join(base, "sessions")
}

// lookup takes the first entry naming the session; files mid-write or
// malformed are skipped, never fatal, and files naming another session are
// not even decoded.
func lookup(dir, sessionID string) (state.HostSession, bool) {
	if dir == "" || sessionID == "" {
		return state.HostSession{}, false
	}
	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return state.HostSession{}, false
	}
	needle := []byte(sessionID)
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil || !bytes.Contains(data, needle) {
			continue
		}
		var entry sessionFile
		if json.Unmarshal(data, &entry) != nil || entry.SessionID != sessionID {
			continue
		}
		return state.HostSession{PID: max(entry.PID, 0), Working: entry.Status == busyStatus}, true
	}
	return state.HostSession{}, false
}
