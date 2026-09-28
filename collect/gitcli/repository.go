package gitcli

import (
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kitsunium/statusline/snapshot"
)

const (
	// statusCodeUntracked is the porcelain code of an untracked file.
	statusCodeUntracked string = "??"
	// minStatusLineLength is the shortest meaningful porcelain line.
	minStatusLineLength int = 2
	// minNumstatParts is the fields of a numstat line: added, removed.
	minNumstatParts int = 2
)

func newRepository() *Repository { return &Repository{} }

// command runs git in dir; a relative or empty dir keeps the daemon's own
// working directory, as the legacy client kept its own.
func command(dir string, args ...string) *exec.Cmd {
	cmd := exec.Command("git", args...)
	if filepath.IsAbs(dir) {
		cmd.Dir = dir
	}
	return cmd
}

func (r *Repository) status(dir string) snapshot.GitStatus {
	out, err := command(dir, "branch", "--show-current").Output()
	// Outside a repository there is no branch and no status
	if err != nil {
		return snapshot.GitStatus{}
	}
	modified, untracked := changeCounts(dir)
	return snapshot.GitStatus{
		Branch:    strings.TrimSpace(string(out)),
		Modified:  modified,
		Untracked: untracked,
		Worktrees: countWorktrees(dir),
	}
}

// countWorktrees counts linked worktrees; an old git or a failure shows none.
func countWorktrees(dir string) int {
	out, err := command(dir, "worktree", "list", "--porcelain").Output()
	if err != nil {
		return 0
	}
	return parseWorktrees(string(out))
}

// parseWorktrees skips the main work tree (the first record) and the
// prunable ones, whose directory is gone: nothing is worked on there.
func parseWorktrees(porcelain string) int {
	count := 0
	for idx, record := range strings.Split(strings.TrimSpace(porcelain), "\n\n") {
		if idx == 0 || strings.TrimSpace(record) == "" {
			continue
		}
		if strings.Contains(record, "\nprunable") {
			continue
		}
		count++
	}
	return count
}

func changeCounts(dir string) (modified, untracked int) {
	out, err := command(dir, "status", "--porcelain").Output()
	if err != nil {
		return 0, 0
	}
	for line := range strings.SplitSeq(string(out), "\n") {
		if len(line) < minStatusLineLength {
			continue
		}
		if line[:minStatusLineLength] == statusCodeUntracked {
			untracked++
		} else {
			modified++
		}
	}
	return modified, untracked
}

func (r *Repository) diffStats(dir string) snapshot.Changes {
	out, err := command(dir, "diff", "--numstat", "HEAD").Output()
	if err != nil {
		return snapshot.Changes{}
	}
	var changes snapshot.Changes
	for line := range strings.SplitSeq(string(out), "\n") {
		if line == "" {
			continue
		}
		added, removed := parseNumstatLine(line)
		changes.Added += added
		changes.Removed += removed
	}
	return changes
}

// parseNumstatLine reads "added removed path"; a binary file shows "-".
func parseNumstatLine(line string) (added, removed int) {
	parts := strings.Fields(line)
	if len(parts) < minNumstatParts {
		return 0, 0
	}
	return digits(parts[0]), digits(parts[1])
}

// digits keeps the decimal digits of a field, as the legacy parser did.
func digits(field string) int {
	if field == "-" {
		return 0
	}
	n := 0
	for _, ch := range field {
		if ch >= '0' && ch <= '9' {
			n = n*10 + int(ch-'0')
		}
	}
	return n
}
