package gitcli_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/kitsunium/statusline/collect/gitcli"
	"github.com/kitsunium/statusline/snapshot"
)

// repo builds a repository with one commit, two modified files, one
// untracked file and one linked worktree.
func repo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root := t.TempDir()
	dir := filepath.Join(root, "repo")
	env := append(os.Environ(),
		"GIT_CONFIG_NOSYSTEM=1", "HOME="+root,
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.invalid",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.invalid")
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir, cmd.Env = dir, env
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	git("init", "-q", "-b", "main")
	write("a.txt", "1\n2\n3\n")
	write("b.txt", "b\n")
	git("add", "-A")
	git("commit", "-q", "-m", "init")
	git("worktree", "add", "-q", "-b", "wt", filepath.Join(root, "wt"))
	git("checkout", "-q", "-b", "feat/x")
	write("a.txt", "1\n")
	write("b.txt", "b\nc\nd\n")
	write("new.txt", "n\n")
	t.Setenv("HOME", root)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	return dir
}

var ctx = context.Background()

// must drops a nil error: the adapter never returns one.
func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func TestStatusAndDiff(t *testing.T) {
	dir := repo(t)
	r := gitcli.NewGit()
	want := snapshot.GitStatus{Branch: "feat/x", Modified: 2, Untracked: 1, Worktrees: 1}
	if got := must(r.Status(ctx, dir)); got != want {
		t.Errorf("Status() = %+v, want %+v", got, want)
	}
	if got, want := must(r.DiffStats(ctx, dir)), (snapshot.Changes{Added: 2, Removed: 2}); got != want {
		t.Errorf("DiffStats() = %+v, want %+v", got, want)
	}
}

func TestOutsideARepository(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(dir))
	r := gitcli.NewGit()
	if got := must(r.Status(ctx, dir)); got != (snapshot.GitStatus{}) {
		t.Errorf("Status() outside a repository = %+v, want zero", got)
	}
	if got := must(r.DiffStats(ctx, dir)); got != (snapshot.Changes{}) {
		t.Errorf("DiffStats() outside a repository = %+v, want zero", got)
	}
}
