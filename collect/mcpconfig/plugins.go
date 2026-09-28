// Package mcp provides the MCP configuration adapter.
package mcpconfig

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kitsunium/statusline/snapshot"
)

// Plugin file locations.
const (
	// installedPluginsPath is the plugin registry, relative to <config>.
	installedPluginsPath string = "plugins/installed_plugins.json"
	// settingsFileName is the settings file in <config> and <project>/.claude.
	settingsFileName string = "settings.json"
	// localSettingsFileName is the uncommitted project settings file.
	localSettingsFileName string = "settings.local.json"
)

// readPluginServers reads the MCP servers of the enabled plugins.
//
// A plugin is enabled when enabledPlugins["<plugin>@<marketplace>"] is true
// in the merged settings (user, then project, then local overriding). Its
// servers come from <installPath>/.mcp.json, wrapped or bare.
//
// Returns:
//   - []snapshot.MCPServer: servers tagged with their plugin, by plugin id order
func (p *reader) readPluginServers() []snapshot.MCPServer {
	// Without a config directory there is no plugin registry
	if p.configDir == "" {
		return []snapshot.MCPServer{}
	}
	enabled := p.enabledPlugins()
	// No plugin enabled, nothing to read
	if len(enabled) == 0 {
		return []snapshot.MCPServer{}
	}
	var registry installedPlugins
	// No registry, no installed plugin
	if !readJSON(filepath.Join(p.configDir, installedPluginsPath), &registry) {
		return []snapshot.MCPServer{}
	}

	ids := make([]string, 0, len(registry.Plugins))
	// Only the enabled plugins are read
	for id := range registry.Plugins {
		// A disabled or unlisted plugin provides nothing
		if enabled[id] {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)

	servers := make([]snapshot.MCPServer, 0, defaultSliceCapacity)
	// Read each enabled plugin's manifest
	for _, id := range ids {
		dir := p.installPath(registry.Plugins[id])
		// An installation without a path cannot be read
		if dir == "" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, projectMCPFileName))
		// Most plugins provide no server
		if err != nil {
			continue
		}
		name, _, _ := strings.Cut(id, "@")
		servers = append(servers, convertServers(parseServers(data, true), name)...)
	}
	return servers
}

// installPath picks the installation that applies to this project.
//
// Params:
//   - installs: installations of one plugin
//
// Returns:
//   - string: install directory, empty when none applies
func (p *reader) installPath(installs []pluginInstall) string {
	// The first user installation, or one scoped to this project, applies
	for _, in := range installs {
		// Another project's installation is not this session's
		if in.ProjectPath == "" || in.ProjectPath == p.projectDir {
			return in.InstallPath
		}
	}
	return ""
}

// enabledPlugins merges the enabledPlugins switches of every settings file.
//
// Returns:
//   - map[string]bool: plugin id to enabled, the more local file winning
func (p *reader) enabledPlugins() map[string]bool {
	paths := []string{filepath.Join(p.configDir, settingsFileName)}
	// The project's own settings apply on top of the user's
	if p.projectDir != "" {
		projectDir := filepath.Join(p.projectDir, claudeConfigDir)
		paths = append(paths,
			filepath.Join(projectDir, settingsFileName),
			filepath.Join(projectDir, localSettingsFileName),
		)
	}
	merged := make(map[string]bool, defaultMapCapacity)
	// Later files override earlier ones, switch by switch
	for _, path := range paths {
		var settings settingsFile
		// A missing or malformed file changes nothing
		if !readJSON(path, &settings) {
			continue
		}
		// Copy every switch the file sets
		for id, on := range settings.EnabledPlugins {
			merged[id] = on
		}
	}
	return merged
}
