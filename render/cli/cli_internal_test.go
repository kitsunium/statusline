package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/kitsunium/statusline/ipc"
	"github.com/kitsunium/statusline/render/show"
	"github.com/kitsunium/statusline/snapshot"
)

type clock struct{}

func (clock) Now(context.Context) (time.Time, error) {
	return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), nil
}

// daemon is the client's view of a daemon, running or not.
type daemon struct {
	running bool
	stopped bool
}

var errNone = errors.New("no daemon")

func (d *daemon) Snapshot(context.Context, ipc.Key) (snapshot.Snapshot, string, error) {
	return snapshot.Snapshot{}, show.OriginNone, nil
}

func (d *daemon) Status(context.Context) (ipc.Status, error) {
	if !d.running {
		return ipc.Status{}, errNone
	}
	return ipc.Status{Version: "v1.2.3", PID: 7, StartedAt: time.Now(), Sessions: 2, BadVersion: "v1.2.4"}, nil
}

func (d *daemon) Stop(context.Context) error {
	if !d.running {
		return errNone
	}
	d.stopped = true
	return nil
}

// run runs the CLI over fake streams and returns what it printed.
func run(d *daemon, args []string, stdin, version string) (string, error) {
	entry := newCLI(show.NewShowStatusLine(d, clock{}), version)
	var out bytes.Buffer
	entry.stdin, entry.stdout = strings.NewReader(stdin), &out
	entry.environ = func() []string { return []string{"COLUMNS=100", "BROKEN"} }
	err := entry.Run(context.Background(), args)
	return out.String(), err
}

func TestVersion(t *testing.T) {
	if out, err := run(&daemon{}, []string{"--version"}, "", "v1.2.3"); out != "statusline v1.2.3\n" || err != nil {
		t.Errorf("--version = %q, %v", out, err)
	}
	if out, _ := run(&daemon{}, []string{"-v"}, "", ""); out != "statusline dev\n" {
		t.Errorf("-v of a development build = %q", out)
	}
}

func TestRenderNeverFails(t *testing.T) {
	for _, args := range [][]string{nil, {"--refresh-usage"}, {"daemon", "status"}} {
		out, err := run(&daemon{}, args, "{not json", "v1")
		if err != nil || strings.Count(out, "\n") != 2 {
			t.Errorf("args %v: %q, %v", args, out, err)
		}
	}
}

func TestEnvironment(t *testing.T) {
	env := environment([]string{"A=1", "B=x=y", "NOEQ", "A=2"})
	if env["A"] != "2" || env["B"] != "x=y" || len(env) != 2 {
		t.Errorf("environment() = %v", env)
	}
}
