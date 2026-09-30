package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Scenario is one synthetic situation the status line is run in: what the
// host pipes on stdin, the files it finds on disk, the repositories it
// queries, the network payloads it would fetch, and the environment.
//
// Every value is synthetic: the repository is public, so no real path, e-mail,
// session id, token or transcript ever appears here.
type Scenario struct {
	// Name is the directory the goldens are written under.
	Name string `json:"name"`
	// Description says what the scenario pins down.
	Description string `json:"description"`
	// Stdin is the payload, a JSON value with placeholders; StdinRaw, when
	// set, is written verbatim instead (malformed payloads).
	Stdin    json.RawMessage `json:"stdin,omitempty"`
	StdinRaw *string         `json:"stdin_raw,omitempty"`
	// Env is added to the sandbox environment for every variant.
	Env map[string]string `json:"env,omitempty"`
	// ConfigDir relocates the configuration directory (CLAUDE_CONFIG_DIR),
	// relative to the sandbox root; empty keeps ~/.claude.
	ConfigDir string `json:"config_dir,omitempty"`
	// Files maps a sandbox-relative path to its content: a JSON string is
	// written as text, an array as JSONL, anything else as JSON.
	Files map[string]json.RawMessage `json:"files,omitempty"`
	// Repos are git repositories created in the sandbox, in order.
	Repos []Repo `json:"repos,omitempty"`
	// Host, when set, starts a stand-in for the host process whose command
	// line and working directory the status line may read.
	Host *Host `json:"host,omitempty"`
	// Usage is the OAuth usage payload; nil means the endpoint answers
	// nothing usable and no cache exists.
	Usage json.RawMessage `json:"usage,omitempty"`
	// UsageAgeSec is how old the usage payload is.
	UsageAgeSec int `json:"usage_age_s,omitempty"`
	// Health is the status page summary; nil means none was ever fetched.
	Health json.RawMessage `json:"health,omitempty"`
	// HealthAgeSec is how old the summary is.
	HealthAgeSec int `json:"health_age_s,omitempty"`
	// Credentials writes a synthetic OAuth token where the host keeps it.
	Credentials bool `json:"credentials,omitempty"`
	// EvenSecond runs each variant at the start of an even wall-clock second:
	// the in-progress task pulses on even seconds.
	EvenSecond bool `json:"even_second,omitempty"`
	// Scope says which binaries a golden binds: "black-box" (both, through
	// the process boundary) or "legacy-only" (the old cache layout is part of
	// the situation, so the new binary is checked by the unit named in Note).
	Scope string `json:"scope,omitempty"`
	// Note explains a legacy-only scope.
	Note string `json:"note,omitempty"`
	// Variants are the environment variations rendered; none means one
	// variant called "default".
	Variants []Variant `json:"variants,omitempty"`
}

// Variant is one environment variation of a scenario.
type Variant struct {
	Name string            `json:"name"`
	Env  map[string]string `json:"env,omitempty"`
}

// Repo is a git repository built in the sandbox.
type Repo struct {
	// Dir is sandbox-relative.
	Dir    string `json:"dir"`
	Branch string `json:"branch"`
	// Commit is committed first, path to content.
	Commit map[string]string `json:"commit"`
	// Modify overwrites tracked files after the commit.
	Modify map[string]string `json:"modify,omitempty"`
	// Untracked adds files after the commit.
	Untracked map[string]string `json:"untracked,omitempty"`
	// Worktrees are linked worktrees, sandbox-relative; a name prefixed with
	// "!" is added then deleted, which leaves a prunable entry.
	Worktrees []string `json:"worktrees,omitempty"`
}

// Host is the stand-in host process.
type Host struct {
	Args []string `json:"args"`
	Cwd  string   `json:"cwd"`
}

// expander resolves the placeholders of one sandbox run.
type expander struct {
	root string
	home string
	pid  int
	now  time.Time
}

var placeholder = regexp.MustCompile(`\{\{([a-z0-9]+)(?::([+-]?[0-9]+))?\}\}`)

// value returns what a placeholder stands for.
func (e expander) value(kind, offset string) (string, bool) {
	off, _ := strconv.Atoi(offset)
	at := e.now.Add(time.Duration(off) * time.Second)
	switch kind {
	case "root":
		return e.root, true
	case "home":
		return e.home, true
	case "pid":
		return strconv.Itoa(e.pid), true
	case "epoch":
		return strconv.FormatInt(at.Unix(), 10), true
	case "epochms":
		return strconv.FormatInt(at.UnixMilli(), 10), true
	case "rfc3339":
		return at.UTC().Format(time.RFC3339Nano), true
	}
	return "", false
}

// text expands every placeholder of a string.
func (e expander) text(s string) string {
	return placeholder.ReplaceAllStringFunc(s, func(m string) string {
		sub := placeholder.FindStringSubmatch(m)
		if v, ok := e.value(sub[1], sub[2]); ok {
			return v
		}
		return m
	})
}

// numeric reports whether a whole-string placeholder yields a JSON number.
func numeric(s string) bool {
	sub := placeholder.FindStringSubmatch(s)
	return sub != nil && sub[0] == s && (sub[1] == "epoch" || sub[1] == "epochms" || sub[1] == "pid")
}

// tree expands the placeholders of a decoded JSON value.
func (e expander) tree(v any) any {
	switch t := v.(type) {
	case string:
		out := e.text(t)
		if numeric(t) {
			return json.Number(out)
		}
		return out
	case []any:
		for i := range t {
			t[i] = e.tree(t[i])
		}
		return t
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			out[e.text(k)] = e.tree(val)
		}
		return out
	}
	return v
}

// json expands a raw JSON document.
func (e expander) json(raw json.RawMessage) ([]byte, error) {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return json.Marshal(e.tree(v))
}

// fileContent renders the content of a Files entry.
func (e expander) fileContent(raw json.RawMessage) ([]byte, error) {
	trimmed := strings.TrimSpace(string(raw))
	switch {
	case strings.HasPrefix(trimmed, `"`):
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return nil, err
		}
		return []byte(e.text(s)), nil
	case strings.HasPrefix(trimmed, "["):
		var lines []json.RawMessage
		if err := json.Unmarshal(raw, &lines); err != nil {
			return nil, err
		}
		var sb strings.Builder
		for _, l := range lines {
			b, err := e.json(l)
			if err != nil {
				return nil, err
			}
			sb.Write(b)
			sb.WriteByte('\n')
		}
		return []byte(sb.String()), nil
	}
	return e.json(raw)
}

// loadScenarios reads every scenario file of a directory, sorted by name.
func loadScenarios(dir string) ([]Scenario, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	out := make([]Scenario, 0, len(paths))
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		var s Scenario
		dec := json.NewDecoder(strings.NewReader(string(data)))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&s); err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		if s.Name == "" {
			s.Name = strings.TrimSuffix(filepath.Base(p), ".json")
		}
		if len(s.Variants) == 0 {
			s.Variants = []Variant{{Name: "default"}}
		}
		if s.Scope == "" {
			s.Scope = "black-box"
		}
		out = append(out, s)
	}
	return out, nil
}
