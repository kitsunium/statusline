// Package mcp provides the MCP configuration adapter.
package mcpconfig

import (
	"encoding/json"
	"os"
)

// userConfigFile represents the ~/.claude.json structure.
// Contains user-level MCP servers and project-specific configs.
type userConfigFile struct {
	// MCPServers at root level (user scope)
	MCPServers map[string]mcpServerConfig `json:"mcpServers"`
	// Projects contains project-specific configurations (local scope)
	Projects map[string]projectConfig `json:"projects"`
}

// projectConfig represents a project's configuration in ~/.claude.json.
// It holds the local-scope servers and the per-project on/off switches.
type projectConfig struct {
	// MCPServers are the local-scope servers
	MCPServers map[string]mcpServerConfig `json:"mcpServers"`
	// DisabledMcpServers lists servers switched off for this project
	DisabledMcpServers []string `json:"disabledMcpServers"`
	// DisabledMcpjsonServers lists project .mcp.json servers rejected here
	DisabledMcpjsonServers []string `json:"disabledMcpjsonServers"`
}

// mcpConfigFile represents the .mcp.json or managed-mcp.json structure.
// Used for project-level and enterprise managed MCP configurations.
type mcpConfigFile struct {
	MCPServers map[string]mcpServerConfig `json:"mcpServers"`
}

// mcpServerConfig represents a single MCP server configuration.
// Only the switch is read: the rest of the entry is the host's business.
type mcpServerConfig struct {
	// Disabled indicates if the server is disabled
	Disabled bool `json:"disabled,omitempty"`
}

// settingsFile is the part of a settings.json the adapter reads.
type settingsFile struct {
	// EnabledPlugins maps "<plugin>@<marketplace>" to its switch
	EnabledPlugins map[string]bool `json:"enabledPlugins"`
}

// installedPlugins is <config>/plugins/installed_plugins.json.
type installedPlugins struct {
	// Plugins maps "<plugin>@<marketplace>" to its installations
	Plugins map[string][]pluginInstall `json:"plugins"`
}

// pluginInstall is one installation of a plugin.
type pluginInstall struct {
	// InstallPath is the directory the plugin was unpacked into
	InstallPath string `json:"installPath"`
	// ProjectPath scopes a project or local installation, empty for user
	ProjectPath string `json:"projectPath"`
}

// readJSON decodes a JSON file.
//
// Params:
//   - path: file to read
//   - dst: value to decode into
//
// Returns:
//   - bool: true when the file was read and decoded
func readJSON(path string, dst any) bool {
	data, err := os.ReadFile(path)
	// An unreadable file is simply absent
	if err != nil {
		return false
	}
	return json.Unmarshal(data, dst) == nil
}

// parseServers decodes an MCP config document.
//
// The document is {"mcpServers": {...}}; with bare set, a document without
// that key may also be the server map itself, as plugin manifests allow.
//
// Params:
//   - data: JSON document
//   - bare: whether a bare server map is accepted
//
// Returns:
//   - map[string]mcpServerConfig: servers declared, nil when malformed
func parseServers(data []byte, bare bool) map[string]mcpServerConfig {
	var doc map[string]json.RawMessage
	// Anything but a JSON object declares nothing
	if json.Unmarshal(data, &doc) != nil {
		return nil
	}
	raw, wrapped := doc["mcpServers"]
	// Without the wrapper only a bare map, when allowed, is left to read
	if !wrapped {
		// Strict documents need the wrapper
		if !bare {
			return nil
		}
		raw = data
	}
	var servers map[string]mcpServerConfig
	// A server map with a malformed entry is rejected whole
	if json.Unmarshal(raw, &servers) != nil {
		return nil
	}
	return servers
}
