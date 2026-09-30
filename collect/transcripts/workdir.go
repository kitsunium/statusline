// Package activity infers where the session is actually working.
package transcripts

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// Transcript scanning constants.
const (
	// tailSize is how much of the transcript end is read. Recent tool calls
	// are all that matter, and a session transcript grows to megabytes.
	dirTailSize int64 = 256 << 10
	// bookkeepingDir holds the session's own transcripts and memory, which
	// say nothing about the project being worked on.
	bookkeepingDir string = ".claude/projects"
)

// cdPattern matches a command that starts by changing directory, and
// gitDirPattern a git invocation aimed at another work tree. Compiled on
// first use: the client links this package but never runs it.
var (
	cdPattern = sync.OnceValue(func() *regexp.Regexp {
		return regexp.MustCompile(`^\s*cd\s+("[^"]+"|'[^']+'|[^\s;&|]+)`)
	})
	gitDirPattern = sync.OnceValue(func() *regexp.Regexp {
		return regexp.MustCompile(`(?:^|[\s;&|])git\s+-C\s+("[^"]+"|'[^']+'|[^\s;&|]+)`)
	})
)

// dirEntry is the part of a transcript line this package reads.
type dirEntry struct {
	Message struct {
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

// dirBlock is one content dirBlock of a message.
type dirBlock struct {
	Type  string    `json:"type"`
	Input toolInput `json:"input"`
}

// toolInput holds the tool arguments that name a location.
type toolInput struct {
	FilePath     string `json:"file_path"`
	NotebookPath string `json:"notebook_path"`
	Path         string `json:"path"`
	Command      string `json:"command"`
}

// dir returns the directory the session is working in.
//
// Claude Code reports the session's working directory, but an agent often
// works elsewhere without moving it: every command runs as "cd X && ...", or
// files are read and edited by absolute path. The last location a tool call
// touched says more, so it wins; the reported directory is the fallback.
// A location inside a git work tree is reported as that tree's root, so the
// segment does not jump between sub-folders of one project.
//
// Params:
//   - transcriptPath: session transcript, may be empty
//   - fallback: directory reported by Claude Code
//
// Returns:
//   - string: directory to display and to run git in
func dir(transcriptPath, fallback string) string {
	// Without a transcript there is nothing better than the reported directory
	if transcriptPath == "" {
		return fallback
	}
	tail, err := readTail(transcriptPath)
	// An unreadable transcript leaves the reported directory in place
	if err != nil {
		return fallback
	}
	lines := bytes.Split(tail, []byte("\n"))
	// Walk back from the most recent line
	for i := len(lines) - 1; i >= 0; i-- {
		// The first existing location found is the most recent one
		if dir := lineDir(lines[i], fallback); dir != "" {
			return dir
		}
	}
	return fallback
}

// readTail reads the end of the transcript, starting at a line boundary.
//
// Params:
//   - path: file to read
//
// Returns:
//   - []byte: the last complete lines
//   - error: any error opening or reading the file
func readTail(path string) ([]byte, error) {
	return tail(path, dirTailSize)
}

// lineDir extracts the most recent working location from one transcript line.
//
// Params:
//   - line: one JSONL record
//   - base: directory relative paths are resolved against
//
// Returns:
//   - string: resolved directory, empty when the line names none
func lineDir(line []byte, base string) string {
	// Only assistant tool calls name locations; skip everything else cheaply
	if !bytes.Contains(line, []byte(`"tool_use"`)) {
		return ""
	}
	var rec dirEntry
	// A line that does not decode names nothing
	if err := json.Unmarshal(line, &rec); err != nil {
		return ""
	}
	var blocks []dirBlock
	// Plain-text content carries no tool call
	if err := json.Unmarshal(rec.Message.Content, &blocks); err != nil {
		return ""
	}
	// Within one message, the last call is the most recent
	for i := len(blocks) - 1; i >= 0; i-- {
		// Text and thinking blocks name no location
		if blocks[i].Type != "tool_use" {
			continue
		}
		// Keep the first candidate that resolves to a real directory
		for _, candidate := range candidates(blocks[i].Input) {
			if dir := resolve(candidate, base); dir != "" {
				return dir
			}
		}
	}
	return ""
}

// candidates lists the locations a tool call names, most specific first.
//
// Params:
//   - in: tool arguments
//
// Returns:
//   - []string: raw paths, possibly relative or starting with ~
func candidates(in toolInput) []string {
	var paths []string
	// File tools name their target directly
	for _, p := range []string{in.FilePath, in.NotebookPath, in.Path} {
		if p != "" {
			paths = append(paths, p)
		}
	}
	// A shell command names its location by changing into it
	if m := cdPattern().FindStringSubmatch(in.Command); m != nil {
		paths = append(paths, unquote(m[1]))
	}
	// Or by pointing git at another work tree
	if m := gitDirPattern().FindStringSubmatch(in.Command); m != nil {
		paths = append(paths, unquote(m[1]))
	}
	return paths
}

// unquote strips one level of shell quoting.
//
// Params:
//   - s: possibly quoted word
//
// Returns:
//   - string: the word without its surrounding quotes
func unquote(s string) string {
	// Only a matching pair of quotes is shell quoting
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1]
	}
	return s
}

// resolve turns a raw path into the directory to report.
//
// Params:
//   - raw: path as written in the tool call
//   - base: directory relative paths are resolved against
//
// Returns:
//   - string: git root or directory, empty when the path cannot be used
func resolve(raw, base string) string {
	home, _ := os.UserHomeDir()
	path := raw
	// Expand the home shorthand the shell would have expanded
	if home != "" && (path == "~" || strings.HasPrefix(path, "~/")) {
		path = filepath.Join(home, strings.TrimPrefix(path, "~"))
	}
	// A relative path is relative to the session's reported directory
	if !filepath.IsAbs(path) {
		path = filepath.Join(base, path)
	}
	path = filepath.Clean(path)
	// The session's own bookkeeping is not a project
	if home != "" && strings.HasPrefix(path, filepath.Join(home, bookkeepingDir)) {
		return ""
	}
	info, err := os.Stat(path)
	// A location that no longer exists cannot be shown or queried
	if err != nil {
		return ""
	}
	// A file is shown through the directory holding it
	if !info.IsDir() {
		path = filepath.Dir(path)
	}
	return gitRoot(path)
}

// gitRoot returns the work tree root holding a directory, or the directory.
//
// Params:
//   - dir: existing directory
//
// Returns:
//   - string: nearest ancestor holding .git, dir itself when there is none
func gitRoot(dir string) string {
	// Climb until a .git dirEntry (directory or worktree file) is found
	for cur := dir; ; cur = filepath.Dir(cur) {
		if _, err := os.Stat(filepath.Join(cur, ".git")); err == nil {
			return cur
		}
		// The filesystem root ends the climb
		if filepath.Dir(cur) == cur {
			return dir
		}
	}
}
