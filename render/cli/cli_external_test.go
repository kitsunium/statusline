package cli_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/kitsunium/statusline/ipc"
	"github.com/kitsunium/statusline/render/cli"
	"github.com/kitsunium/statusline/render/ctl"
	"github.com/kitsunium/statusline/render/port"
	"github.com/kitsunium/statusline/render/show"
	"github.com/kitsunium/statusline/snapshot"
)

type clock struct{}

func (clock) Now() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }

type daemon struct {
	running bool
	stopped bool
}

func (d *daemon) Snapshot(context.Context, ipc.Key) (snapshot.Snapshot, port.SnapshotOrigin) {
	return snapshot.Snapshot{}, port.OriginNone
}

func (d *daemon) Status(context.Context) (ipc.Status, error) {
	if !d.running {
		return ipc.Status{}, context.DeadlineExceeded
	}
	return ipc.Status{Version: "v1.2.3", PID: 7, StartedAt: time.Now(), Sessions: 2, BadVersion: "v1.2.4"}, nil
}

func (d *daemon) Stop(context.Context) error {
	if !d.running {
		return context.DeadlineExceeded
	}
	d.stopped = true
	return nil
}

func run(d *daemon, args []string, stdin, version string) (string, int) {
	entry := cli.New(cli.Deps{
		Show:   show.New(show.Deps{Clock: clock{}, Snapshots: d}),
		Status: ctl.NewStatus(ctl.Deps{Daemon: d}),
		Stop:   ctl.NewStop(ctl.Deps{Daemon: d}),
	})
	var out bytes.Buffer
	code := entry.Run(cli.RunInput{Args: append([]string{"statusline"}, args...), Stdin: strings.NewReader(stdin), Stdout: &out,
		Getenv: func(string) string { return "" }, Version: version})
	return out.String(), code
}

func TestVersion(t *testing.T) {
	if out, code := run(&daemon{}, []string{"--version"}, "", "v1.2.3"); out != "statusline v1.2.3\n" || code != 0 {
		t.Errorf("--version = %q, %d", out, code)
	}
	if out, _ := run(&daemon{}, []string{"-v"}, "", ""); out != "statusline dev\n" {
		t.Errorf("-v of a development build = %q", out)
	}
}

func TestDaemonStatus(t *testing.T) {
	out, code := run(&daemon{running: true}, []string{"daemon", "status"}, "", "v1")
	if code != 0 || !strings.Contains(out, "daemon v1.2.3 pid 7") || !strings.Contains(out, "2 session(s)") || !strings.Contains(out, "v1.2.4") {
		t.Errorf("status = %q, %d", out, code)
	}
	if out, code := run(&daemon{}, []string{"daemon", "status"}, "", "v1"); code != 1 || !strings.Contains(out, "no daemon") {
		t.Errorf("status without daemon = %q, %d", out, code)
	}
}

func TestDaemonStop(t *testing.T) {
	d := &daemon{running: true}
	if out, _ := run(d, []string{"daemon", "stop"}, "", "v1"); !d.stopped || !strings.Contains(out, "stopped") {
		t.Errorf("stop = %q, stopped %v", out, d.stopped)
	}
	if out, code := run(&daemon{}, []string{"daemon", "stop"}, "", "v1"); code != 0 || !strings.Contains(out, "no daemon") {
		t.Errorf("stop without daemon = %q, %d", out, code)
	}
}

func TestRenderNeverFails(t *testing.T) {
	for _, args := range [][]string{nil, {"--refresh-usage"}, {"daemon", "what"}} {
		out, code := run(&daemon{}, args, "{not json", "v1")
		if code != 0 || strings.Count(out, "\n") != 2 {
			t.Errorf("args %v: %q, %d", args, out, code)
		}
	}
}
