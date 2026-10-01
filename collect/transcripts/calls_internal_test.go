package transcripts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// base is the frozen "now" of every test.
var base = time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)

// at formats a moment relative to base as a transcript timestamp.
func at(d time.Duration) string {
	return base.Add(d).Format(time.RFC3339Nano)
}

// use is an assistant record calling tool name with id, d relative to base.
func use(t *testing.T, id, name string, d time.Duration) string {
	t.Helper()
	return record(t, "assistant", d, map[string]any{"type": "tool_use", "id": id, "name": name, "input": map[string]any{}})
}

// result is a user record answering call id, d relative to base.
func result(t *testing.T, id string, d time.Duration) string {
	t.Helper()
	return record(t, "user", d, map[string]any{"type": "tool_result", "tool_use_id": id, "content": "ok"})
}

func record(t *testing.T, kind string, d time.Duration, b map[string]any) string {
	t.Helper()
	out, err := json.Marshal(map[string]any{
		"type": kind, "timestamp": at(d),
		"message": map[string]any{"role": kind, "content": []any{b}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func writeLines(t *testing.T, path string, lines ...string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

// fixture builds a provider on a temporary session.
func fixture(t *testing.T) (*calls, string) {
	t.Helper()
	root := t.TempDir()
	main := filepath.Join(root, "projects", "p", "sess.jsonl")
	p := &calls{
		transcriptPath: main,
		subagentDir:    filepath.Join(root, "projects", "p", "sess", "subagents"),
		agentsFile:     filepath.Join(root, "kodflow", "sessions", "sess", "agents.json"),
		now:            func() time.Time { return base },
	}
	return p, main
}

// busyCase is one main transcript and the servers it must light.
type busyCase struct {
	name  string
	lines func(t *testing.T) []string
	want  string
}

// runBusyCases runs each case on a fresh session.
func runBusyCases(t *testing.T, tests []busyCase) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, main := fixture(t)
			writeLines(t, main, tt.lines(t)...)
			if got := strings.Join(p.busy(), ","); got != tt.want {
				t.Errorf("Busy() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBusyMainTranscript(t *testing.T) {
	runBusyCases(t, []busyCase{
		{
			name: "call in flight",
			lines: func(t *testing.T) []string {
				return []string{use(t, "a", "mcp__github__get_me", -5*time.Second)}
			},
			want: "github",
		},
		{
			name: "returned within the linger window",
			lines: func(t *testing.T) []string {
				return []string{use(t, "a", "mcp__github__get_me", -3*time.Second), result(t, "a", -1500*time.Millisecond)}
			},
			want: "github",
		},
		{
			name: "returned and expired",
			lines: func(t *testing.T) []string {
				return []string{use(t, "a", "mcp__github__get_me", -9*time.Second), result(t, "a", -3*time.Second)}
			},
			want: "",
		},
		{
			name: "plugin server keeps its tool-name key",
			lines: func(t *testing.T) []string {
				return []string{use(t, "a", "mcp__plugin_kodflow-hooks_tasks__task_list", -time.Second)}
			},
			want: "plugin_kodflow-hooks_tasks",
		},
	})
}

func TestBusyIgnoresNoise(t *testing.T) {
	runBusyCases(t, []busyCase{
		{
			name: "built-in tools and malformed names are ignored",
			lines: func(t *testing.T) []string {
				return []string{use(t, "a", "Bash", -time.Second), use(t, "b", "mcp__nameonly", -time.Second)}
			},
			want: "",
		},
		{
			name: "a stale call whose result was lost is ignored",
			lines: func(t *testing.T) []string {
				return []string{use(t, "a", "mcp__github__get_me", -time.Hour)}
			},
			want: "",
		},
		{
			name: "malformed lines are skipped",
			lines: func(t *testing.T) []string {
				return []string{`{"message":{"content":[{"type":"tool_use","name":"mcp__torn`, `not json "mcp__x__y"`,
					`{"message":{"content":"text with \"mcp__a__b\""}}`, use(t, "a", "mcp__gitlab__get_project", -time.Second)}
			},
			want: "gitlab",
		},
		{
			name: "several calls, one returned long ago",
			lines: func(t *testing.T) []string {
				return []string{
					use(t, "a", "mcp__github__x", -time.Minute), use(t, "b", "mcp__context7__y", -time.Minute),
					result(t, "a", -50*time.Second), use(t, "c", "mcp__clangd__z", -time.Second),
				}
			},
			want: "clangd,context7",
		},
	})
}

func TestBusyTruncatedTail(t *testing.T) {
	p, main := fixture(t)
	// The call is pushed out of the tail; only its result and a new call stay
	filler := `{"type":"user","message":{"content":"` + strings.Repeat("x", int(callsTailSize)) + `"}}`
	writeLines(t, main,
		use(t, "old", "mcp__github__x", -3*time.Second), filler,
		result(t, "old", -time.Second), use(t, "new", "mcp__gitlab__y", -time.Second))
	if got := strings.Join(p.busy(), ","); got != "gitlab" {
		t.Errorf("Busy() = %q, want gitlab: a result without its call is ignored", got)
	}
}

func TestBusySubagents(t *testing.T) {
	p, main := fixture(t)
	writeLines(t, main, use(t, "m", "Agent", -time.Minute))
	started := base.Add(-time.Minute).Unix()
	stopped := base.Unix()
	reg, _ := json.Marshal(map[string]any{"agents": map[string]any{
		"arun":     map[string]any{"started": started},
		"astopped": map[string]any{"started": started, "stopped": stopped},
		"astale":   map[string]any{"started": base.Add(-13 * time.Hour).Unix()},
		"../evil":  map[string]any{"started": started},
		"anofile":  map[string]any{"started": started},
	}})
	writeLines(t, p.agentsFile, string(reg))
	writeLines(t, filepath.Join(p.subagentDir, "agent-arun.jsonl"), use(t, "s", "mcp__playwright__browser_click", -time.Second))
	writeLines(t, filepath.Join(p.subagentDir, "agent-astopped.jsonl"), use(t, "s", "mcp__github__x", -time.Second))
	writeLines(t, filepath.Join(p.subagentDir, "agent-astale.jsonl"), use(t, "s", "mcp__gitlab__x", -time.Second))
	if got := strings.Join(p.busy(), ","); got != "playwright" {
		t.Errorf("Busy() = %q, want playwright (only the running subagent)", got)
	}

	writeLines(t, p.agentsFile, `{"agents":`)
	if got := p.busy(); len(got) != 0 {
		t.Errorf("malformed registry: Busy() = %v, want nothing", got)
	}
}

func TestBusyWithoutTranscript(t *testing.T) {
	if got := newCalls("", "sess", time.Now()).busy(); got != nil {
		t.Errorf("no transcript: %v", got)
	}
	p, _ := fixture(t)
	if got := p.busy(); len(got) != 0 {
		t.Errorf("missing transcript: %v", got)
	}
}

func TestNewCallsPaths(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv(configDirEnv, cfg)
	p := newCalls("/x/projects/p/abc.jsonl", "abc", time.Now())
	if p.subagentDir != "/x/projects/p/abc/subagents" {
		t.Errorf("subagentDir = %q", p.subagentDir)
	}
	if p.agentsFile != filepath.Join(cfg, "kodflow", "sessions", "abc", "agents.json") {
		t.Errorf("agentsFile = %q", p.agentsFile)
	}
	if q := newCalls("/x/a.jsonl", "", time.Now()); q.subagentDir != "" || q.agentsFile != "" {
		t.Errorf("no session id: %+v", q)
	}
}

func TestServerKey(t *testing.T) {
	tests := map[string]string{
		"mcp__github__get_me":                        "github",
		"mcp__plugin_kodflow-hooks_tasks__task_list": "plugin_kodflow-hooks_tasks",
		"mcp__claude_ai_Claude_Docs__guide":          "claude_ai_Claude_Docs",
		"mcp__nameonly":                              "",
		"Read":                                       "",
	}
	for name, want := range tests {
		if got := serverKey(name); got != want {
			t.Errorf("serverKey(%q) = %q, want %q", name, got, want)
		}
	}
}
