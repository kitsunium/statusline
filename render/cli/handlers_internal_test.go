package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/kitsunium/sdk/framework/kit"

	"github.com/kitsunium/statusline/ipc"
	"github.com/kitsunium/statusline/render/show"
	"github.com/kitsunium/statusline/snapshot"
)

type clock struct{}

func (clock) Now(context.Context) (time.Time, error) {
	return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), nil
}

type noDaemon struct{}

func (noDaemon) Snapshot(context.Context, ipc.Key) (snapshot.Snapshot, string, error) {
	return snapshot.Snapshot{}, show.OriginNone, nil
}

// run runs the command over buffers and returns what it printed.
func run(args []string, stdin, version string) (string, int) {
	Wire(show.NewShowStatusLine(noDaemon{}, clock{}), version)
	var out, errb bytes.Buffer
	code := line(context.Background(), args, kit.Stdio{In: strings.NewReader(stdin), Out: &out, Err: &errb})
	return out.String() + errb.String(), code
}

func TestVersion(t *testing.T) {
	if out, code := run([]string{"--version"}, "", "v1.2.3"); out != "statusline v1.2.3\n" || code != 0 {
		t.Errorf("--version = %q, %d", out, code)
	}
	if out, _ := run([]string{"-v"}, "", ""); out != "statusline dev\n" {
		t.Errorf("-v of a development build = %q", out)
	}
}

func TestRenderNeverFails(t *testing.T) {
	for _, args := range [][]string{nil, {"--refresh-usage"}, {"serve"}, {"what", "ever"}} {
		out, code := run(args, "{not json", "v1")
		if code != 0 || strings.Count(out, "\n") != 2 {
			t.Errorf("args %v: %q, %d", args, out, code)
		}
	}
}

func TestEnvironment(t *testing.T) {
	env := environment([]string{"A=1", "B=x=y", "NOEQ", "A=2"})
	if env["A"] != "2" || env["B"] != "x=y" || len(env) != 2 {
		t.Errorf("environment() = %v", env)
	}
}
