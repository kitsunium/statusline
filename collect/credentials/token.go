package credentials

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	// keychainService is the macOS keychain item the host writes.
	keychainService string = "Claude Code-credentials"
	// credentialsFileName is the credentials file in a config directory.
	credentialsFileName string = ".credentials.json"
	// configDirEnv relocates the host's configuration directory.
	configDirEnv string = "CLAUDE_CONFIG_DIR"
	// keychainTimeout bounds the security(1) call.
	keychainTimeout time.Duration = 2 * time.Second
)

// errNoToken says no source holds a token: the account simply has no API
// enrichment.
var errNoToken = errors.New("statusline_no_token: no OAuth token found")

// credentialsFile is the part of the host's credentials read here.
type credentialsFile struct {
	OAuth struct {
		AccessToken string `json:"accessToken"`
	} `json:"claudeAiOauth"`
}

func (s *Store) token(ctx context.Context) (string, error) {
	if runtime.GOOS == "darwin" {
		if token := keychainToken(ctx); token != "" {
			return token, nil
		}
	}
	for _, path := range s.candidates() {
		if token := fileToken(path); token != "" {
			return token, nil
		}
	}
	return "", errNoToken
}

// candidates lists the credentials files, the relocated directory first.
func (s *Store) candidates() []string {
	var paths []string
	if dir := s.getenv(configDirEnv); dir != "" {
		paths = append(paths, filepath.Join(dir, credentialsFileName))
	}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".claude", credentialsFileName))
	}
	return paths
}

// keychainToken asks the macOS keychain; empty on any failure.
func keychainToken(ctx context.Context) string {
	ctx, cancel := context.WithTimeout(ctx, keychainTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "security", "find-generic-password", "-s", keychainService, "-w").Output()
	if err != nil {
		return ""
	}
	return decode(out)
}

// fileToken reads one credentials file; empty on any failure.
func fileToken(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return decode(data)
}

func decode(data []byte) string {
	var creds credentialsFile
	if json.Unmarshal(data, &creds) != nil {
		return ""
	}
	return strings.TrimSpace(creds.OAuth.AccessToken)
}
