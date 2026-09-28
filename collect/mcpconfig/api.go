// Package mcpconfig reads the MCP servers a session can reach from the
// host's configuration files, its command line and its enabled plugins.
//
// Exported API of design/domains/collect.yaml (collect/component/mcpconfig).
// This file stands in for the shells kit generates (api_gen.go) until
// `kit gen` is available: every exported symbol only delegates to its
// unexported twin.
package mcpconfig

import "github.com/kitsunium/statusline/snapshot"

// Reader implements collect/port/mcp-config@v1.
type Reader struct {
	configDir   string
	userConfigs []string
	managedPath string
	procDir     string
}

// New resolves the configuration locations from the environment.
func New() *Reader { return newBase() }

// Servers returns the servers of a project, the first source naming a
// server winning: managed > command line > local > project > user > plugin.
func (r *Reader) Servers(projectDir string, hostPID int) snapshot.MCPServers {
	return r.servers(projectDir, hostPID)
}
