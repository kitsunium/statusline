package transcripts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// toolLine builds one assistant transcript line carrying a tool call.
func toolLine(t *testing.T, name string, input map[string]string) string {
	t.Helper()
	rec := map[string]any{
		"type": "assistant",
		"message": map[string]any{"content": []map[string]any{
			{"type": "text", "text": "working"},
			{"type": "tool_use", "name": name, "input": input},
		}},
	}
	out, err := json.Marshal(rec)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// writeTranscript writes lines as a JSONL transcript and returns its path.
func writeTranscript(t *testing.T, lines ...string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "session.jsonl")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDir(t *testing.T) {
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o700); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(repo, "internal", "pkg")
	if err := os.MkdirAll(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(sub, "x.go")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	plain := t.TempDir()
	fallback := t.TempDir()

	tests := []struct {
		name  string
		lines []string
		want  string
	}{
		{name: "edited file resolves to its git root",
			lines: []string{toolLine(t, "Edit", map[string]string{"file_path": file})}, want: repo},
		{name: "leading cd in a command",
			lines: []string{toolLine(t, "Bash", map[string]string{"command": "cd '" + plain + "' && ls"})}, want: plain},
		{name: "git -C in a command",
			lines: []string{toolLine(t, "Bash", map[string]string{"command": "git -C " + sub + " status"})}, want: repo},
		{name: "most recent call wins",
			lines: []string{
				toolLine(t, "Read", map[string]string{"file_path": file}),
				toolLine(t, "Bash", map[string]string{"command": "cd " + plain}),
			}, want: plain},
		{name: "vanished location falls back to an older one",
			lines: []string{
				toolLine(t, "Read", map[string]string{"file_path": file}),
				toolLine(t, "Bash", map[string]string{"command": "cd " + filepath.Join(plain, "gone")}),
			}, want: repo},
		{name: "commands without a location are skipped",
			lines: []string{
				toolLine(t, "Glob", map[string]string{"path": plain}),
				toolLine(t, "Bash", map[string]string{"command": "go test ./..."}),
			}, want: plain},
		{name: "no tool call keeps the fallback",
			lines: []string{`{"type":"user","message":{"content":"hello"}}`, `not json`}, want: fallback},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dir(writeTranscript(t, tt.lines...), fallback); got != tt.want {
				t.Errorf("dir() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDirWithoutTranscript(t *testing.T) {
	if got := dir("", "/fallback"); got != "/fallback" {
		t.Errorf("dir(\"\") = %q, want the fallback", got)
	}
	if got := dir("/does/not/exist.jsonl", "/fallback"); got != "/fallback" {
		t.Errorf("dir(missing) = %q, want the fallback", got)
	}
}

func TestDirSkipsSessionBookkeeping(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	memory := filepath.Join(home, ".claude", "projects", "p", "memory")
	if err := os.MkdirAll(memory, 0o700); err != nil {
		t.Fatal(err)
	}
	work := t.TempDir()
	path := writeTranscript(t,
		toolLine(t, "Bash", map[string]string{"command": "cd " + work}),
		toolLine(t, "Write", map[string]string{"file_path": filepath.Join(memory, "note.md")}),
	)
	if got := dir(path, "/fallback"); got != work {
		t.Errorf("dir() = %q, want %q", got, work)
	}
}

func TestReadTailStartsOnALineBoundary(t *testing.T) {
	long := strings.Repeat("x", int(dirTailSize))
	path := writeTranscript(t, long, `{"last":true}`)
	data, err := readTail(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), `{"last":true}`) {
		t.Errorf("tail starts with %q, want the last complete line", string(data[:min(20, len(data))]))
	}
}
