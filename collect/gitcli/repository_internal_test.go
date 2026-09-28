package gitcli

import (
	"testing"
)

func TestParseNumstatLine(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		wantAdded   int
		wantRemoved int
	}{
		{name: "normal line", line: "10	5	filename.go", wantAdded: 10, wantRemoved: 5},
		{name: "binary file", line: "-	-	image.png", wantAdded: 0, wantRemoved: 0},
		{name: "empty line", line: "", wantAdded: 0, wantRemoved: 0},
		{name: "large numbers", line: "100	200	bigfile.go", wantAdded: 100, wantRemoved: 200},
		{name: "only added", line: "50	0	newfile.go", wantAdded: 50, wantRemoved: 0},
		{name: "malformed single field", line: "10", wantAdded: 0, wantRemoved: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			added, removed := parseNumstatLine(tt.line)
			if added != tt.wantAdded || removed != tt.wantRemoved {
				t.Errorf("parseNumstatLine(%q) = (%d, %d), want (%d, %d)", tt.line, added, removed, tt.wantAdded, tt.wantRemoved)
			}
		})
	}
}

func TestParseWorktrees(t *testing.T) {
	tests := []struct {
		name      string
		porcelain string
		want      int
	}{
		{name: "main only", porcelain: "worktree /r\nHEAD abc\nbranch refs/heads/main\n", want: 0},
		{name: "two linked", porcelain: "worktree /r\nHEAD a\nbranch refs/heads/main\n\nworktree /r/.claude/worktrees/a\nHEAD b\nbranch refs/heads/a\n\nworktree /tmp/b\nHEAD c\ndetached\n", want: 2},
		{name: "prunable is not counted", porcelain: "worktree /r\nHEAD a\n\nworktree /gone\nHEAD b\ndetached\nprunable gitdir file points to non-existent location\n", want: 0},
		{name: "empty", porcelain: "", want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseWorktrees(tt.porcelain); got != tt.want {
				t.Errorf("parseWorktrees() = %d, want %d", got, tt.want)
			}
		})
	}
}
