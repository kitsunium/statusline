// Package mcp provides the MCP configuration adapter.
package mcpconfig

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kitsunium/statusline/snapshot"
)

// Command-line flags the host accepts for MCP servers.
const (
	// mcpConfigFlag names a config file or an inline JSON config; repeatable
	// and variadic.
	mcpConfigFlag string = "--mcp-config"
	// strictFlag restricts the session to the command-line configs.
	strictFlag string = "--strict-mcp-config"
)

// commandLine is what the host's command line says about MCP servers.
type commandLine struct {
	servers []snapshot.MCPServer
	strict  bool
}

// readCommandLine reads the MCP servers given on the host's command line.
//
// The host's pid comes from the session registry; its argv is read from
// /proc/<pid>/cmdline. Without /proc (non-Linux) or a pid, nothing is read.
//
// Returns:
//   - commandLine: servers in declaration order and the strict switch
func (p *reader) readCommandLine() commandLine {
	// Without a pid source or /proc there is no command line to read
	if p.pid == nil || p.procDir == "" {
		return commandLine{}
	}
	pid := p.pid()
	// An unknown host names no process
	if pid <= 0 {
		return commandLine{}
	}
	procDir := filepath.Join(p.procDir, strconv.Itoa(pid))
	data, err := os.ReadFile(filepath.Join(procDir, "cmdline"))
	// A vanished or unreadable process declares nothing
	if err != nil {
		return commandLine{}
	}
	// Relative config paths resolve against the host's working directory
	cwd, err := os.Readlink(filepath.Join(procDir, "cwd"))
	if err != nil {
		cwd = p.projectDir
	}
	args := strings.Split(string(bytes.TrimRight(data, "\x00")), "\x00")
	return parseCommandLine(args, cwd)
}

// parseCommandLine extracts the MCP configs from an argv.
//
// --mcp-config takes one or more values up to the next flag, and may repeat;
// --mcp-config=value is accepted too. Each value is a JSON document when it
// starts with "{", a file path otherwise. The first config naming a server
// wins, matching the order the host was given them.
//
// Params:
//   - args: argv, program name first
//   - cwd: directory relative paths resolve against
//
// Returns:
//   - commandLine: servers and the strict switch
func parseCommandLine(args []string, cwd string) commandLine {
	var out commandLine
	var values []string
	collecting := false
	// Skip the program name and walk the arguments once
	for _, arg := range args[min(1, len(args)):] {
		// A flag ends any value list in progress
		if strings.HasPrefix(arg, "-") {
			collecting = false
		}
		// Sort the argument by what it is
		switch {
		case arg == strictFlag:
			out.strict = true
		case arg == mcpConfigFlag:
			collecting = true
		case strings.HasPrefix(arg, mcpConfigFlag+"="):
			values = append(values, strings.TrimPrefix(arg, mcpConfigFlag+"="))
		case collecting:
			values = append(values, arg)
		}
	}

	seen := make(map[string]bool, defaultMapCapacity)
	// Merge the configs in the order they were given
	for _, value := range values {
		// Keep only names no earlier config declared
		for _, s := range convertServers(loadConfigValue(value, cwd), "") {
			// The first config naming a server wins
			if seen[s.Name] {
				continue
			}
			seen[s.Name] = true
			out.servers = append(out.servers, s)
		}
	}
	return out
}

// loadConfigValue resolves one --mcp-config value.
//
// Params:
//   - value: inline JSON or a file path
//   - cwd: directory relative paths resolve against
//
// Returns:
//   - map[string]mcpServerConfig: servers declared, nil when unusable
func loadConfigValue(value, cwd string) map[string]mcpServerConfig {
	trimmed := strings.TrimSpace(value)
	// An inline document is parsed as is
	if strings.HasPrefix(trimmed, "{") {
		return parseServers([]byte(trimmed), false)
	}
	path := trimmed
	// A relative path is the host's, not ours
	if !filepath.IsAbs(path) && cwd != "" {
		path = filepath.Join(cwd, path)
	}
	data, err := os.ReadFile(path)
	// A missing file declares nothing
	if err != nil {
		return nil
	}
	return parseServers(data, false)
}
