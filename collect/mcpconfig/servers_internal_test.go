package mcpconfig

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/kitsunium/statusline/snapshot"
)

// write creates a file and its parent directories.
func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// fixture is an isolated machine: a config dir, a project, a /proc.
type fixture struct {
	root, config, project, proc string
}

// newFixture builds an empty machine and a provider reading only from it.
func newFixture(t *testing.T) (*fixture, *reader) {
	t.Helper()
	root := t.TempDir()
	f := &fixture{
		root:    root,
		config:  filepath.Join(root, "config"),
		project: filepath.Join(root, "project"),
		proc:    filepath.Join(root, "proc"),
	}
	p := &reader{
		projectDir:  f.project,
		configDir:   f.config,
		userConfigs: []string{filepath.Join(f.config, userConfigFileName)},
		managedPath: filepath.Join(root, "managed", managedMCPFileName),
		procDir:     f.proc,
	}
	return f, p
}

// cmdline installs a host process with the given argv under the fake /proc.
func (f *fixture) cmdline(t *testing.T, p *reader, pid int, args ...string) {
	t.Helper()
	write(t, filepath.Join(f.proc, strconv.Itoa(pid), "cmdline"), strings.Join(args, "\x00")+"\x00")
	p.pid = func() int { return pid }
}

// names renders servers as "name" or "name(off)" for compact assertions.
func names(servers snapshot.MCPServers) string {
	out := make([]string, 0, len(servers))
	for _, s := range servers {
		n := s.Name
		if s.Plugin != "" {
			n += "@" + s.Plugin
		}
		if !s.Enabled {
			n += "(off)"
		}
		out = append(out, n)
	}
	return strings.Join(out, ",")
}

func TestNewPaths(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv(configDirEnv, cfg)
	p := newBase()
	if p.configDir != cfg {
		t.Errorf("configDir = %q, want %q", p.configDir, cfg)
	}
	if len(p.userConfigs) != 1 || p.userConfigs[0] != filepath.Join(cfg, userConfigFileName) {
		t.Errorf("userConfigs = %v, want the relocated .claude.json only", p.userConfigs)
	}

	t.Setenv(configDirEnv, "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	p = newBase()
	want := []string{filepath.Join(home, userConfigFileName), filepath.Join(home, claudeConfigDir, userConfigFileName)}
	if strings.Join(p.userConfigs, "|") != strings.Join(want, "|") {
		t.Errorf("userConfigs = %v, want %v", p.userConfigs, want)
	}
	if p.managedPath != "" && !filepath.IsAbs(p.managedPath) {
		t.Errorf("managedPath = %q, want absolute", p.managedPath)
	}
}

func TestProjectConfigPaths(t *testing.T) {
	if got := (&reader{}).projectConfigPaths(); got != nil {
		t.Errorf("no project: %v, want nil", got)
	}
	if got := (&reader{projectDir: "/w"}).projectConfigPaths(); len(got) != 2 || filepath.Base(got[0]) != projectMCPFileName {
		t.Errorf("paths = %v, want .mcp.json then mcp.json", got)
	}
}

func TestServersEmptyMachine(t *testing.T) {
	_, p := newFixture(t)
	if got := p.servers(); got == nil || len(got) != 0 {
		t.Errorf("Servers() = %v, want empty non-nil", got)
	}
}

// sourceCase is one machine setup and the servers it must produce.
type sourceCase struct {
	name  string
	setup func(t *testing.T, f *fixture, p *reader)
	want  string
}

// runSourceCases runs each case on a fresh machine.
func runSourceCases(t *testing.T, tests []sourceCase) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, p := newFixture(t)
			tt.setup(t, f, p)
			if got := names(p.servers()); got != tt.want {
				t.Errorf("Servers() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestServersConfigFiles(t *testing.T) {
	runSourceCases(t, []sourceCase{
		{
			name: "user scope",
			setup: func(t *testing.T, f *fixture, _ *reader) {
				write(t, filepath.Join(f.config, userConfigFileName), `{"mcpServers":{"GitKraken":{}}}`)
			},
			want: "GitKraken",
		},
		{
			name: "local scope",
			setup: func(t *testing.T, f *fixture, _ *reader) {
				write(t, filepath.Join(f.config, userConfigFileName), `{"projects":{"`+f.project+`":{"mcpServers":{"loc":{}}}}}`)
			},
			want: "loc",
		},
		{
			name: "project .mcp.json",
			setup: func(t *testing.T, f *fixture, _ *reader) {
				write(t, filepath.Join(f.project, ".mcp.json"), `{"mcpServers":{"proj":{"disabled":true}}}`)
			},
			want: "proj(off)",
		},
		{
			name: "project mcp.json fallback",
			setup: func(t *testing.T, f *fixture, _ *reader) {
				write(t, filepath.Join(f.project, "mcp.json"), `{"mcpServers":{"undotted":{}}}`)
			},
			want: "undotted",
		},
		{
			name: "project malformed .mcp.json falls back",
			setup: func(t *testing.T, f *fixture, _ *reader) {
				write(t, filepath.Join(f.project, ".mcp.json"), `{"mcpServers":`)
				write(t, filepath.Join(f.project, "mcp.json"), `{"mcpServers":{"undotted":{}}}`)
			},
			want: "undotted",
		},
		{
			name: "managed",
			setup: func(t *testing.T, _ *fixture, p *reader) {
				write(t, p.managedPath, `{"mcpServers":{"corp":{}}}`)
			},
			want: "corp",
		},
	})
}

func TestServersCommandLine(t *testing.T) {
	runSourceCases(t, []sourceCase{
		{
			name: "command line file",
			setup: func(t *testing.T, f *fixture, p *reader) {
				cfg := filepath.Join(f.root, "mcp.json")
				write(t, cfg, `{"mcpServers":{"context7":{},"github":{}}}`)
				f.cmdline(t, p, 42, "claude", "--model", "default", "--mcp-config", cfg, "--resume", "abc")
			},
			want: "context7,github",
		},
		{
			name: "command line inline JSON",
			setup: func(t *testing.T, f *fixture, p *reader) {
				f.cmdline(t, p, 42, "claude", `--mcp-config={"mcpServers":{"inline":{}}}`)
			},
			want: "inline",
		},
		{
			name: "command line repeated and variadic",
			setup: func(t *testing.T, f *fixture, p *reader) {
				write(t, filepath.Join(f.root, "a.json"), `{"mcpServers":{"a":{},"shared":{}}}`)
				write(t, filepath.Join(f.root, "b.json"), `{"mcpServers":{"b":{},"shared":{"disabled":true}}}`)
				f.cmdline(t, p, 42, "claude", "--mcp-config", filepath.Join(f.root, "a.json"), `{"mcpServers":{"c":{}}}`,
					"--verbose", "--mcp-config", filepath.Join(f.root, "b.json"))
			},
			want: "a,shared,c,b",
		},
	})
}

func TestServersCommandLineEdges(t *testing.T) {
	runSourceCases(t, []sourceCase{
		{
			name: "command line relative path resolves against the host cwd",
			setup: func(t *testing.T, f *fixture, p *reader) {
				write(t, filepath.Join(f.root, "work", "rel.json"), `{"mcpServers":{"rel":{}}}`)
				f.cmdline(t, p, 42, "claude", "--mcp-config", "rel.json")
				if err := os.Symlink(filepath.Join(f.root, "work"), filepath.Join(f.proc, "42", "cwd")); err != nil {
					t.Fatal(err)
				}
			},
			want: "rel",
		},
		{
			name: "command line bad values fail open",
			setup: func(t *testing.T, f *fixture, p *reader) {
				f.cmdline(t, p, 42, "claude", "--mcp-config", "/nope.json", `{"mcpServers":`, `{"servers":{"x":{}}}`)
			},
			want: "",
		},
	})
}

func TestServersPlugins(t *testing.T) {
	runSourceCases(t, []sourceCase{
		{
			name: "plugin enabled",
			setup: func(t *testing.T, f *fixture, _ *reader) {
				installPlugin(t, f, "kodflow-hooks@kodflow", `{"mcpServers":{"tasks":{}}}`)
				write(t, filepath.Join(f.config, settingsFileName), `{"enabledPlugins":{"kodflow-hooks@kodflow":true}}`)
			},
			want: "tasks@kodflow-hooks",
		},
		{
			name: "plugin bare server map",
			setup: func(t *testing.T, f *fixture, _ *reader) {
				installPlugin(t, f, "bare@m", `{"srv":{"command":"x"}}`)
				write(t, filepath.Join(f.config, settingsFileName), `{"enabledPlugins":{"bare@m":true}}`)
			},
			want: "srv@bare",
		},
		{
			name: "plugin disabled",
			setup: func(t *testing.T, f *fixture, _ *reader) {
				installPlugin(t, f, "kodflow-hooks@kodflow", `{"mcpServers":{"tasks":{}}}`)
				write(t, filepath.Join(f.config, settingsFileName), `{"enabledPlugins":{"kodflow-hooks@kodflow":false}}`)
			},
			want: "",
		},
		{
			name: "plugin not in enabledPlugins",
			setup: func(t *testing.T, f *fixture, _ *reader) {
				installPlugin(t, f, "kodflow-hooks@kodflow", `{"mcpServers":{"tasks":{}}}`)
				write(t, filepath.Join(f.config, settingsFileName), `{"enabledPlugins":{}}`)
			},
			want: "",
		},
	})
}

func TestServersPluginScope(t *testing.T) {
	runSourceCases(t, []sourceCase{
		{
			name: "plugin disabled by project local settings",
			setup: func(t *testing.T, f *fixture, _ *reader) {
				installPlugin(t, f, "p@m", `{"mcpServers":{"s":{}}}`)
				write(t, filepath.Join(f.config, settingsFileName), `{"enabledPlugins":{"p@m":true}}`)
				write(t, filepath.Join(f.project, ".claude", settingsFileName), `{"enabledPlugins":{"p@m":false}}`)
				write(t, filepath.Join(f.project, ".claude", localSettingsFileName), `{"enabledPlugins":{"p@m":true}}`)
			},
			want: "s@p",
		},
		{
			name: "plugin without .mcp.json",
			setup: func(t *testing.T, f *fixture, _ *reader) {
				installPlugin(t, f, "kodflow-workflow@kodflow", "")
				write(t, filepath.Join(f.config, settingsFileName), `{"enabledPlugins":{"kodflow-workflow@kodflow":true}}`)
			},
			want: "",
		},
		{
			name: "plugin installed for another project",
			setup: func(t *testing.T, f *fixture, _ *reader) {
				dir := filepath.Join(f.root, "plugin")
				write(t, filepath.Join(dir, ".mcp.json"), `{"mcpServers":{"s":{}}}`)
				write(t, filepath.Join(f.config, installedPluginsPath),
					`{"version":2,"plugins":{"p@m":[{"scope":"project","projectPath":"/elsewhere","installPath":"`+dir+`"}]}}`)
				write(t, filepath.Join(f.config, settingsFileName), `{"enabledPlugins":{"p@m":true}}`)
			},
			want: "",
		},
	})
}

// installPlugin registers a plugin whose manifest is body (none when empty).
func installPlugin(t *testing.T, f *fixture, id, body string) {
	t.Helper()
	dir := filepath.Join(f.root, "plugins", id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if body != "" {
		write(t, filepath.Join(dir, ".mcp.json"), body)
	}
	write(t, filepath.Join(f.config, installedPluginsPath),
		`{"version":2,"plugins":{"`+id+`":[{"scope":"user","installPath":"`+dir+`"}]}}`)
}

// everywhere declares server "dup" in every source, each with its own twist.
func everywhere(t *testing.T, f *fixture, p *reader, strict bool) {
	t.Helper()
	write(t, p.managedPath, `{"mcpServers":{"managed":{}}}`)
	args := []string{"claude", `--mcp-config={"mcpServers":{"dup":{},"cli":{}}}`}
	if strict {
		args = append(args, "--strict-mcp-config")
	}
	f.cmdline(t, p, 7, args...)
	write(t, filepath.Join(f.config, userConfigFileName), `{
		"mcpServers":{"dup":{"disabled":true},"user":{}},
		"projects":{"`+f.project+`":{"mcpServers":{"dup":{"disabled":true},"local":{}}}}}`)
	write(t, filepath.Join(f.project, ".mcp.json"), `{"mcpServers":{"dup":{"disabled":true},"project":{}}}`)
	installPlugin(t, f, "plug@m", `{"mcpServers":{"dup":{"disabled":true},"plugin":{}}}`)
	write(t, filepath.Join(f.config, settingsFileName), `{"enabledPlugins":{"plug@m":true}}`)
}

func TestServersPrecedence(t *testing.T) {
	f, p := newFixture(t)
	everywhere(t, f, p, false)
	want := "managed,cli,dup,local,project,user,plugin@plug"
	if got := names(p.servers()); got != want {
		t.Errorf("Servers() = %q, want %q (command line wins dup)", got, want)
	}

	// Managed beats the command line
	write(t, p.managedPath, `{"mcpServers":{"dup":{"disabled":true}}}`)
	if got := names(p.servers()); !strings.HasPrefix(got, "dup(off),cli,") {
		t.Errorf("Servers() = %q, want managed dup first", got)
	}

	// Local beats project, project beats user, user beats plugin
	f, p = newFixture(t)
	write(t, filepath.Join(f.config, userConfigFileName), `{"mcpServers":{"a":{"disabled":true},"b":{}},
		"projects":{"`+f.project+`":{"mcpServers":{"a":{}}}}}`)
	write(t, filepath.Join(f.project, ".mcp.json"), `{"mcpServers":{"a":{"disabled":true},"b":{"disabled":true}}}`)
	installPlugin(t, f, "plug@m", `{"mcpServers":{"b":{},"c":{}}}`)
	write(t, filepath.Join(f.config, settingsFileName), `{"enabledPlugins":{"plug@m":true}}`)
	if got, want := names(p.servers()), "a,b(off),c@plug"; got != want {
		t.Errorf("Servers() = %q, want %q", got, want)
	}
}

func TestServersStrict(t *testing.T) {
	f, p := newFixture(t)
	everywhere(t, f, p, true)
	if got, want := names(p.servers()), "managed,cli,dup"; got != want {
		t.Errorf("strict Servers() = %q, want %q", got, want)
	}
}

func TestServersDisabledLists(t *testing.T) {
	f, p := newFixture(t)
	write(t, filepath.Join(f.config, userConfigFileName), `{"mcpServers":{"u":{},"keep":{}},
		"projects":{"`+f.project+`":{"disabledMcpServers":["u","plugin:plug:tasks"],"disabledMcpjsonServers":["proj"]}}}`)
	write(t, filepath.Join(f.project, ".mcp.json"), `{"mcpServers":{"proj":{},"ok":{}}}`)
	installPlugin(t, f, "plug@m", `{"mcpServers":{"tasks":{}}}`)
	write(t, filepath.Join(f.config, settingsFileName), `{"enabledPlugins":{"plug@m":true}}`)
	if got, want := names(p.servers()), "ok,proj(off),keep,u(off),tasks@plug(off)"; got != want {
		t.Errorf("Servers() = %q, want %q", got, want)
	}
}

func TestServersMalformedFilesFailOpen(t *testing.T) {
	f, p := newFixture(t)
	write(t, p.managedPath, `not json`)
	write(t, filepath.Join(f.config, userConfigFileName), `{"mcpServers":[1,2]}`)
	write(t, filepath.Join(f.project, ".mcp.json"), `{"mcpServers":{"x":"y"}}`)
	write(t, filepath.Join(f.config, settingsFileName), `{"enabledPlugins":"yes"}`)
	write(t, filepath.Join(f.config, installedPluginsPath), `{"plugins":`)
	f.cmdline(t, p, 9, "claude", "--mcp-config", "{bad")
	if got := p.servers(); len(got) != 0 {
		t.Errorf("Servers() = %v, want nothing from malformed files", got)
	}

	// A malformed plugin manifest or registry entry skips only that plugin
	f, p = newFixture(t)
	installPlugin(t, f, "bad@m", `{"mcpServers":{"x":1}}`)
	write(t, filepath.Join(f.config, settingsFileName), `{"enabledPlugins":{"bad@m":true}}`)
	if got := p.servers(); len(got) != 0 {
		t.Errorf("Servers() = %v, want nothing from a malformed manifest", got)
	}
}

func TestServersGlobalConfigFallback(t *testing.T) {
	f, p := newFixture(t)
	first := filepath.Join(f.root, "home", userConfigFileName)
	p.userConfigs = []string{first, filepath.Join(f.config, userConfigFileName)}
	write(t, filepath.Join(f.config, userConfigFileName), `{"mcpServers":{"second":{}}}`)
	if got := names(p.servers()); got != "second" {
		t.Errorf("missing first candidate: %q, want second", got)
	}
	write(t, first, `{"mcpServers":{"first":{}}}`)
	if got := names(p.servers()); got != "first" {
		t.Errorf("first candidate present: %q, want first", got)
	}
}

func TestCommandLineWithoutProc(t *testing.T) {
	f, p := newFixture(t)
	f.cmdline(t, p, 42, "claude", `--mcp-config={"mcpServers":{"x":{}}}`)
	p.procDir = ""
	if got := p.readCommandLine(); len(got.servers) != 0 || got.strict {
		t.Errorf("no /proc: %+v, want nothing", got)
	}
	p.procDir = filepath.Join(f.root, "missing")
	if got := p.readCommandLine(); len(got.servers) != 0 {
		t.Errorf("missing /proc entry: %+v, want nothing", got)
	}
	p.procDir = f.proc
	p.pid = func() int { return 0 }
	if got := p.readCommandLine(); len(got.servers) != 0 {
		t.Errorf("unknown pid: %+v, want nothing", got)
	}
	p.pid = nil
	if got := p.readCommandLine(); len(got.servers) != 0 {
		t.Errorf("no pid source: %+v, want nothing", got)
	}
}

func TestParseCommandLineEdges(t *testing.T) {
	if got := parseCommandLine(nil, ""); len(got.servers) != 0 || got.strict {
		t.Errorf("empty argv: %+v", got)
	}
	// A value list stops at the next flag; the program name is never a value
	got := parseCommandLine([]string{`{"mcpServers":{"prog":{}}}`, "--mcp-config", `{"mcpServers":{"a":{}}}`, "--x", `{"mcpServers":{"b":{}}}`}, "")
	if names(got.servers) != "a" {
		t.Errorf("servers = %q, want a", names(got.servers))
	}
}

func TestParseServers(t *testing.T) {
	tests := []struct {
		name string
		data string
		bare bool
		want int
	}{
		{name: "wrapped", data: `{"mcpServers":{"a":{},"b":{}}}`, want: 2},
		{name: "bare refused", data: `{"a":{}}`, want: 0},
		{name: "bare accepted", data: `{"a":{}}`, bare: true, want: 1},
		{name: "not an object", data: `[1]`, bare: true, want: 0},
		{name: "bad entry", data: `{"a":3}`, bare: true, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := len(parseServers([]byte(tt.data), tt.bare)); got != tt.want {
				t.Errorf("len = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestConvertServers(t *testing.T) {
	if got := convertServers(nil, ""); got == nil || len(got) != 0 {
		t.Errorf("nil map: %v, want empty non-nil", got)
	}
	got := convertServers(map[string]mcpServerConfig{"b": {Disabled: true}, "a": {}}, "plug")
	if names(got) != "a@plug,b@plug(off)" {
		t.Errorf("convertServers = %q", names(got))
	}
}

func TestServersSourceTags(t *testing.T) {
	f, p := newFixture(t)
	everywhere(t, f, p, false)
	want := map[string]snapshot.MCPSource{
		"managed": snapshot.MCPSourceManaged,
		"cli":     snapshot.MCPSourceCLI,
		"dup":     snapshot.MCPSourceCLI,
		"local":   snapshot.MCPSourceLocal,
		"project": snapshot.MCPSourceProject,
		"user":    snapshot.MCPSourceUser,
		"plugin":  snapshot.MCPSourcePlugin,
	}
	got := p.servers()
	if len(got) != len(want) {
		t.Fatalf("Servers() = %q, want %d servers", names(got), len(want))
	}
	for _, s := range got {
		if s.Source != want[s.Name] {
			t.Errorf("%s: Source = %q, want %q (the scope that won the name)", s.Name, s.Source, want[s.Name])
		}
	}
}
