// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package mcpconfig

import (
	"context"

	"github.com/kitsunium/statusline/snapshot"
)

// config is the Config adapter's own state: the configuration locations,
// fixed for the daemon's life.
type config struct {
	configDir   string
	userConfigs []string
	managedPath string
	procDir     string
}

// newConfig resolves the configuration locations from the environment.
func newConfig() *Config {
	return &Config{*newBase()}
}

// servers reads one project's servers.
func (a *Config) servers(_ context.Context, projectDir string, hostPID int) (snapshot.MCPServers, error) {
	return newReader(&a.config, projectDir, hostPID).servers(), nil
}
