// Package mcp provides the MCP configuration adapter.
package mcpconfig

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"

	"github.com/kitsunium/statusline/snapshot"
)

// Config file names and paths.
const (
	// configDirEnv relocates the whole configuration directory.
	configDirEnv string = "CLAUDE_CONFIG_DIR"
	// claudeConfigDir is the Claude config directory name.
	claudeConfigDir string = ".claude"
	// userConfigFileName is the user-level Claude config file.
	userConfigFileName string = ".claude.json"
	// projectMCPFileName is the project-level MCP config file (dotted).
	projectMCPFileName string = ".mcp.json"
	// projectMCPFallbackFileName is the fallback MCP config file (undotted).
	projectMCPFallbackFileName string = "mcp.json"
	// managedMCPFileName is the enterprise managed MCP config file.
	managedMCPFileName string = "managed-mcp.json"
	// managedPathLinux is the Linux enterprise config directory.
	managedPathLinux string = "/etc/claude-code"
	// managedPathMacOS is the macOS enterprise config directory.
	managedPathMacOS string = "/Library/Application Support/ClaudeCode"
	// procRoot is where Linux exposes each process's command line.
	procRoot string = "/proc"
	// defaultMapCapacity is the default capacity for server tracking map.
	defaultMapCapacity int = 16
	// defaultSliceCapacity is the default capacity for server list.
	defaultSliceCapacity int = 8
)

// reader reads the MCP servers of one project for one call.
//
// Every source is a small file read: the config files, the enabled plugins'
// manifests and, on Linux, the host's command line under /proc. Nothing is
// executed and nothing touches the network.
type reader struct {
	projectDir  string
	configDir   string
	userConfigs []string
	managedPath string
	procDir     string
	pid         func() int
}

// newReader locates the configuration of one project.
//
// Params:
//   - base: the configuration locations fixed for the daemon's life
//   - projectDir: the project directory path
//   - hostPID: the host process running the session, 0 when unknown
//
// Returns:
//   - *reader: reader for that project
func newReader(base *config, projectDir string, hostPID int) *reader {
	return &reader{
		projectDir:  projectDir,
		configDir:   base.configDir,
		userConfigs: base.userConfigs,
		managedPath: base.managedPath,
		procDir:     base.procDir,
		pid:         func() int { return hostPID },
	}
}

// newBase resolves the configuration locations from the environment.
//
// Returns:
//   - *config: locations of the global config, the managed file and /proc
func newBase() *config {
	p := &config{managedPath: managedConfigPath()}
	home, _ := os.UserHomeDir()
	p.configDir = os.Getenv(configDirEnv)
	// A relocated config directory holds the global config file too
	if p.configDir != "" {
		p.userConfigs = []string{filepath.Join(p.configDir, userConfigFileName)}
	} else if home != "" {
		p.configDir = filepath.Join(home, claudeConfigDir)
		p.userConfigs = []string{
			filepath.Join(home, userConfigFileName),
			filepath.Join(p.configDir, userConfigFileName),
		}
	}
	// /proc only exists on Linux; elsewhere the command line stays unread
	if runtime.GOOS == "linux" {
		p.procDir = procRoot
	}
	return p
}

// servers returns the MCP servers the session can reach.
//
// A name is drawn once, from the first source that declares it:
// managed > command line > local > project > user > plugin. With
// --strict-mcp-config only managed and command-line servers count.
// disabledMcpServers (and disabledMcpjsonServers for the project file)
// in the global config's entry for this project mark servers disabled.
//
// Returns:
//   - []snapshot.MCPServer: list of MCP server configurations
func (p *reader) servers() []snapshot.MCPServer {
	global := p.readGlobalConfig()
	local := global.Projects[p.projectDir]
	cli := p.readCommandLine()

	sources := [][]snapshot.MCPServer{
		snapshot.WithSource(p.readManagedConfig(), snapshot.MCPSourceManaged),
		snapshot.WithSource(cli.servers, snapshot.MCPSourceCLI),
	}
	// Strict mode ignores every configured scope but the managed one
	if !cli.strict {
		project := p.readProjectConfig()
		markDisabled(project, local.DisabledMcpjsonServers)
		sources = append(sources,
			snapshot.WithSource(convertServers(local.MCPServers, ""), snapshot.MCPSourceLocal),
			snapshot.WithSource(project, snapshot.MCPSourceProject),
			snapshot.WithSource(convertServers(global.MCPServers, ""), snapshot.MCPSourceUser),
			snapshot.WithSource(p.readPluginServers(), snapshot.MCPSourcePlugin),
		)
	}

	seen := make(map[string]bool, defaultMapCapacity)
	servers := make([]snapshot.MCPServer, 0, defaultSliceCapacity)
	// Walk the sources in precedence order: the first to name a server wins
	for _, source := range sources {
		// Keep only the names no stronger source already declared
		for _, s := range source {
			// A weaker source never overrides a stronger one
			if seen[s.Name] {
				continue
			}
			seen[s.Name] = true
			servers = append(servers, s)
		}
	}
	markDisabled(servers, local.DisabledMcpServers)
	return servers
}

// managedConfigPath returns the path to enterprise managed MCP config.
//
// Returns:
//   - string: platform-specific path to managed-mcp.json
func managedConfigPath() string {
	var basePath string
	// Select path based on platform
	switch runtime.GOOS {
	// macOS enterprise path
	case "darwin":
		basePath = managedPathMacOS
	// Linux enterprise path
	case "linux":
		basePath = managedPathLinux
	// Other platforms not supported
	default:
		return ""
	}
	return filepath.Join(basePath, managedMCPFileName)
}

// projectConfigPaths returns paths to project MCP config files.
// Returns dotted (.mcp.json) first, then undotted (mcp.json) as fallback.
//
// Returns:
//   - []string: paths to check in order
func (p *reader) projectConfigPaths() []string {
	// No project, no project file
	if p.projectDir == "" {
		return nil
	}
	return []string{
		filepath.Join(p.projectDir, projectMCPFileName),
		filepath.Join(p.projectDir, projectMCPFallbackFileName),
	}
}

// readGlobalConfig reads the global config (~/.claude.json).
//
// The first candidate that parses wins: $CLAUDE_CONFIG_DIR/.claude.json when
// the directory is relocated, else ~/.claude.json then ~/.claude/.claude.json.
//
// Returns:
//   - userConfigFile: parsed config, empty when none is readable
func (p *reader) readGlobalConfig() userConfigFile {
	// Try each candidate location in turn
	for _, path := range p.userConfigs {
		var config userConfigFile
		// An unreadable or malformed candidate gives way to the next one
		if readJSON(path, &config) {
			return config
		}
	}
	return userConfigFile{}
}

// readProjectConfig reads MCP servers from project MCP config file.
// Tries .mcp.json first, then falls back to mcp.json (undotted).
//
// Returns:
//   - []snapshot.MCPServer: list of MCP servers from project config
func (p *reader) readProjectConfig() []snapshot.MCPServer {
	// Take the first project file that parses
	for _, path := range p.projectConfigPaths() {
		var config mcpConfigFile
		// A missing or malformed file gives way to the fallback
		if !readJSON(path, &config) {
			continue
		}
		return convertServers(config.MCPServers, "")
	}
	return []snapshot.MCPServer{}
}

// readManagedConfig reads MCP servers from enterprise managed config.
//
// Returns:
//   - []snapshot.MCPServer: list of MCP servers from managed-mcp.json
func (p *reader) readManagedConfig() []snapshot.MCPServer {
	var config mcpConfigFile
	// No managed path, or no readable managed file, declares nothing
	if p.managedPath == "" || !readJSON(p.managedPath, &config) {
		return []snapshot.MCPServer{}
	}
	return convertServers(config.MCPServers, "")
}

// markDisabled marks the listed servers disabled.
//
// A plugin server may be listed under its qualified name
// plugin:<plugin>:<server>, as the host writes it.
//
// Params:
//   - servers: servers to update in place
//   - names: names listed as disabled
func markDisabled(servers []snapshot.MCPServer, names []string) {
	// Nothing listed, nothing to do
	if len(names) == 0 {
		return
	}
	off := make(map[string]bool, len(names))
	// Index the disabled names for a single pass over the servers
	for _, name := range names {
		off[name] = true
	}
	// Disable each server listed by plain or qualified name
	for i := range servers {
		s := &servers[i]
		// Only a listed server changes state
		if off[s.Name] || (s.Plugin != "" && off["plugin:"+s.Plugin+":"+s.Name]) {
			s.Enabled = false
		}
	}
}

// convertServers converts a map of server configs to MCPServers slice.
//
// Params:
//   - servers: map of server name to config
//   - plugin: plugin providing the servers, empty for a config file
//
// Returns:
//   - []snapshot.MCPServer: slice of MCP servers, sorted by name
func convertServers(servers map[string]mcpServerConfig, plugin string) []snapshot.MCPServer {
	// Check if servers map is empty
	if len(servers) == 0 {
		return []snapshot.MCPServer{}
	}

	names := make([]string, 0, len(servers))
	// Extract keys for a deterministic order
	for name := range servers {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]snapshot.MCPServer, 0, len(servers))
	// Convert map to slice in sorted order
	for _, name := range names {
		result = append(result, snapshot.MCPServer{
			Name:    name,
			Enabled: !servers[name].Disabled,
			Plugin:  plugin,
		})
	}
	return result
}
