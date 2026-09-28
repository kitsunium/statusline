package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Flavour says how the network payloads reach the binary under test.
type Flavour string

const (
	// Legacy seeds the on-disk caches of kodflow/status-line: that binary
	// renders from them and never fetches while they are fresh.
	Legacy Flavour = "legacy"
	// Kit serves the payloads over a local HTTP endpoint the new binary is
	// pointed at (STATUSLINE_USAGE_URL, STATUSLINE_HEALTH_URL), and waits for
	// its daemon to be warm before reading the line.
	Kit Flavour = "kit"
)

// syntheticToken is the OAuth token written for scenarios that need one.
const syntheticToken = "sk-ant-oat01-SYNTHETIC-PARITY-TOKEN"

// Sandbox is one scenario materialised on disk.
type Sandbox struct {
	root    string
	home    string
	env     []string
	stdin   []byte
	host    *exec.Cmd
	server  *http.Server
	flavour Flavour
	extra   []string
	exp     expander
}

// buildSandbox materialises a scenario under a fresh temporary root.
func buildSandbox(s Scenario, flavour Flavour, now time.Time) (*Sandbox, error) {
	root, err := os.MkdirTemp("", "parity-")
	if err != nil {
		return nil, err
	}
	root, _ = filepath.EvalSymlinks(root)
	sb := &Sandbox{root: root, home: filepath.Join(root, "home"), flavour: flavour}
	for _, d := range []string{sb.home, filepath.Join(root, "run"), filepath.Join(root, "tmp")} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return nil, err
		}
	}
	sb.exp = expander{root: root, home: sb.home, now: now}

	// The host stand-in first: its pid is a placeholder in the files
	if s.Host != nil {
		if err := sb.startHost(*s.Host); err != nil {
			sb.Close()
			return nil, err
		}
	}
	gitEnv := sb.baseEnv()
	for _, r := range s.Repos {
		if err := buildRepo(filepath.Join(root, r.Dir), r, gitEnv, root); err != nil {
			sb.Close()
			return nil, fmt.Errorf("repo %s: %w", r.Dir, err)
		}
	}
	for rel, raw := range s.Files {
		content, err := sb.exp.fileContent(raw)
		if err != nil {
			sb.Close()
			return nil, fmt.Errorf("file %s: %w", rel, err)
		}
		path := filepath.Join(root, sb.exp.text(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			sb.Close()
			return nil, err
		}
		if err := os.WriteFile(path, content, 0o600); err != nil {
			sb.Close()
			return nil, err
		}
	}
	// A cached usage payload means a token existed when it was fetched: the
	// kit flavour, which fetches instead of reading a cache, needs that token
	if s.Credentials || (flavour == Kit && len(s.Usage) > 0) {
		cred := fmt.Sprintf(`{"claudeAiOauth":{"accessToken":%q}}`, syntheticToken)
		dir := filepath.Join(sb.home, ".claude")
		if err := os.MkdirAll(dir, 0o700); err != nil {
			sb.Close()
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(dir, ".credentials.json"), []byte(cred), 0o600); err != nil {
			sb.Close()
			return nil, err
		}
	}
	if err := sb.network(s, now); err != nil {
		sb.Close()
		return nil, err
	}

	switch {
	case s.StdinRaw != nil:
		sb.stdin = []byte(sb.exp.text(*s.StdinRaw))
	case len(s.Stdin) > 0:
		if sb.stdin, err = sb.exp.json(s.Stdin); err != nil {
			sb.Close()
			return nil, fmt.Errorf("stdin: %w", err)
		}
	}

	sb.env = sb.baseEnv()
	if s.ConfigDir != "" {
		sb.env = append(sb.env, "CLAUDE_CONFIG_DIR="+filepath.Join(root, s.ConfigDir))
	}
	for k, v := range s.Env {
		sb.env = append(sb.env, k+"="+sb.exp.text(v))
	}
	return sb, nil
}

// baseEnv is the closed environment every run starts from: nothing of the
// caller's leaks in, and every network access fails fast through a dead
// proxy unless a local endpoint is named explicitly.
func (sb *Sandbox) baseEnv() []string {
	return []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + sb.home,
		"XDG_RUNTIME_DIR=" + filepath.Join(sb.root, "run"),
		"TMPDIR=" + filepath.Join(sb.root, "tmp"),
		"LANG=C.UTF-8",
		"HTTPS_PROXY=http://127.0.0.1:9",
		"HTTP_PROXY=http://127.0.0.1:9",
		"NO_PROXY=127.0.0.1",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=Parity",
		"GIT_AUTHOR_EMAIL=parity@example.invalid",
		"GIT_COMMITTER_NAME=Parity",
		"GIT_COMMITTER_EMAIL=parity@example.invalid",
		"GIT_AUTHOR_DATE=2026-01-01T00:00:00Z",
		"GIT_COMMITTER_DATE=2026-01-01T00:00:00Z",
		"STATUS_LINE_NO_SELF_UPDATE=1",
		"STATUSLINE_NO_SELF_UPDATE=1",
	}
}

// startHost starts the host stand-in: a shell sleeping with the scenario's
// arguments after its own, so /proc/<pid>/cmdline carries them.
func (sb *Sandbox) startHost(h Host) error {
	args := []string{"-c", "sleep 120", "host"}
	for _, a := range h.Args {
		args = append(args, sb.exp.text(a))
	}
	cmd := exec.Command("/bin/sh", args...)
	cmd.Dir = filepath.Join(sb.root, h.Cwd)
	if err := os.MkdirAll(cmd.Dir, 0o700); err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	sb.host = cmd
	sb.exp.pid = cmd.Process.Pid
	return nil
}

// network makes the usage and health payloads reachable the way the
// flavour expects them.
func (sb *Sandbox) network(s Scenario, now time.Time) error {
	uid := strconv.Itoa(os.Getuid())
	switch sb.flavour {
	case Legacy:
		run := filepath.Join(sb.root, "run")
		if len(s.Usage) > 0 {
			if err := seed(filepath.Join(run, "status-line-usage-"+uid, "usage.json"), sb.exp, s.Usage, now.Add(-time.Duration(s.UsageAgeSec)*time.Second)); err != nil {
				return err
			}
		}
		if len(s.Health) > 0 {
			if err := seed(filepath.Join(run, "status-line-health-"+uid, "summary.json"), sb.exp, s.Health, now.Add(-time.Duration(s.HealthAgeSec)*time.Second)); err != nil {
				return err
			}
		}
		return nil
	case Kit:
		usage, err := optionalJSON(sb.exp, s.Usage)
		if err != nil {
			return err
		}
		health, err := optionalJSON(sb.exp, s.Health)
		if err != nil {
			return err
		}
		mux := http.NewServeMux()
		serve := func(body []byte) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				if body == nil {
					http.Error(w, "none", http.StatusServiceUnavailable)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write(body)
			}
		}
		mux.HandleFunc("/usage", serve(usage))
		mux.HandleFunc("/health", serve(health))
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return err
		}
		sb.server = &http.Server{Handler: mux, ReadHeaderTimeout: time.Second}
		go func() { _ = sb.server.Serve(ln) }()
		base := "http://" + ln.Addr().String()
		sb.extra = append(sb.extra, "STATUSLINE_USAGE_URL="+base+"/usage", "STATUSLINE_HEALTH_URL="+base+"/health")
		return nil
	}
	return fmt.Errorf("unknown flavour %q", sb.flavour)
}

// optionalJSON expands a payload, nil when absent.
func optionalJSON(e expander, raw json.RawMessage) ([]byte, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	return e.json(raw)
}

// seed writes a cache file with the given modification time.
func seed(path string, e expander, raw json.RawMessage, mtime time.Time) error {
	body, err := e.json(raw)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(path, body, 0o600); err != nil {
		return err
	}
	return os.Chtimes(path, mtime, mtime)
}

// buildRepo creates one repository with a single deterministic commit.
func buildRepo(dir string, r Repo, env []string, root string) error {
	git := func(args ...string) error {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, out)
		}
		return nil
	}
	write := func(files map[string]string) error {
		for rel, content := range files {
			p := filepath.Join(dir, rel)
			if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
				return err
			}
			if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
				return err
			}
		}
		return nil
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	branch := r.Branch
	if branch == "" {
		branch = "main"
	}
	if err := git("init", "-q", "-b", "main"); err != nil {
		return err
	}
	if err := write(r.Commit); err != nil {
		return err
	}
	if err := git("add", "-A"); err != nil {
		return err
	}
	if err := git("commit", "-q", "-m", "init"); err != nil {
		return err
	}
	for _, wt := range r.Worktrees {
		prune := strings.HasPrefix(wt, "!")
		path := filepath.Join(root, strings.TrimPrefix(wt, "!"))
		if err := git("worktree", "add", "-q", "-b", "wt-"+filepath.Base(path), path); err != nil {
			return err
		}
		if prune {
			if err := os.RemoveAll(path); err != nil {
				return err
			}
		}
	}
	if branch != "main" {
		if err := git("checkout", "-q", "-b", branch); err != nil {
			return err
		}
	}
	if err := write(r.Modify); err != nil {
		return err
	}
	return write(r.Untracked)
}

// Result is what one run of the binary produced.
type Result struct {
	Stdout []byte
	Stderr []byte
	Exit   int
	Took   time.Duration
}

// run executes the binary once with a variant's environment.
func (sb *Sandbox) run(bin string, v Variant, args ...string) (Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = sb.home
	cmd.Env = append(append(append([]string{}, sb.env...), sb.extra...), variantEnv(sb.exp, v)...)
	cmd.Stdin = bytes.NewReader(sb.stdin)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	start := time.Now()
	err := cmd.Run()
	took := time.Since(start)
	res := Result{Stdout: out.Bytes(), Stderr: errb.Bytes(), Took: took}
	if ee, ok := err.(*exec.ExitError); ok {
		res.Exit = ee.ExitCode()
		err = nil
	}
	return res, err
}

// variantEnv expands a variant's environment.
func variantEnv(e expander, v Variant) []string {
	out := make([]string, 0, len(v.Env))
	for k, val := range v.Env {
		out = append(out, k+"="+e.text(val))
	}
	return out
}

// Close stops the host stand-in and the endpoint, and removes the sandbox.
func (sb *Sandbox) Close() {
	if sb.host != nil && sb.host.Process != nil {
		_ = sb.host.Process.Kill()
		_, _ = sb.host.Process.Wait()
	}
	if sb.server != nil {
		_ = sb.server.Close()
	}
	_ = os.RemoveAll(sb.root)
}
