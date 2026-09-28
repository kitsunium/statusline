package daemon_test

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kitsunium/statusline/collect/daemon"
	"github.com/kitsunium/statusline/collect/gather"
	"github.com/kitsunium/statusline/collect/refresh"
	"github.com/kitsunium/statusline/collect/state"
	"github.com/kitsunium/statusline/collect/update"
	"github.com/kitsunium/statusline/ipc"
	"github.com/kitsunium/statusline/snapshot"
)

// clock returns t0, then jumps by jump on every later call.
type clock struct {
	t0    time.Time
	jump  time.Duration
	calls atomic.Int64
}

func (c *clock) Now() time.Time {
	n := c.calls.Add(1) - 1
	return c.t0.Add(time.Duration(n) * c.jump)
}

type collector struct{ reg *state.Registry }

func (c collector) Execute(_ context.Context, in gather.CollectInput) (gather.CollectOutput, error) {
	c.reg.Touch(in.Key, time.Now())
	return gather.CollectOutput{Snapshot: snapshot.Snapshot{WorkDir: in.Key.SessionDir}, Collected: true}, nil
}

type refresher struct{}

func (refresher) Execute(context.Context, refresh.RefreshInput) (refresh.RefreshOutput, error) {
	return refresh.RefreshOutput{}, nil
}

type latest struct{}

func (latest) Execute(context.Context, refresh.LatestInput) (refresh.LatestOutput, error) {
	return refresh.LatestOutput{Network: state.Network{UsageStatus: 429}}, nil
}

type updater struct{ out update.UpdateOutput }

func (u updater) Execute(context.Context, update.UpdateInput) (update.UpdateOutput, error) {
	return u.out, nil
}

type updates struct{}

func (updates) LoadUpdate() (state.Update, error) { return state.Update{BadVersion: "v0.0.9"}, nil }
func (updates) SaveUpdate(state.Update) error     { return nil }

// instance places the socket under a short path: a Unix socket path is
// limited to about a hundred bytes.
func instance(t *testing.T) ipc.Instance {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "sl")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return ipc.Locate(ipc.LocateInput{RuntimeDir: dir, UID: os.Getuid(), ConfigDir: "/c", Executable: "/e"})
}

func start(t *testing.T, inst ipc.Instance, clk *clock, up update.UpdateOutput, wait bool) <-chan int {
	t.Helper()
	reg := state.NewRegistry()
	d := daemon.New(daemon.Deps{
		Clock: clk, Registry: reg, Collect: collector{reg}, Refresh: refresher{},
		Latest: latest{}, Update: updater{up}, Updates: updates{},
	})
	done := make(chan int, 1)
	go func() {
		done <- d.Run(daemon.RunInput{Version: "v1.0.0", Executable: "/e", Instance: inst, Getenv: func(string) string { return "" }})
	}()
	if !wait {
		return done
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(inst.Socket); err == nil {
			return done
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("the daemon never listened")
	return nil
}

// ask sends one request and returns the daemon's hello and answer.
func ask(t *testing.T, inst ipc.Instance, req ipc.Request) (ipc.Hello, ipc.Response) {
	t.Helper()
	conn, err := net.Dial("unix", inst.Socket)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	if err := ipc.WriteFrame(conn, ipc.Hello{Protocol: ipc.Protocol, Version: "v1.0.0", PID: os.Getpid()}); err != nil {
		t.Fatal(err)
	}
	var hello ipc.Hello
	if err := ipc.ReadFrame(conn, &hello); err != nil {
		t.Fatal(err)
	}
	if err := ipc.WriteFrame(conn, req); err != nil {
		t.Fatal(err)
	}
	var resp ipc.Response
	if err := ipc.ReadFrame(conn, &resp); err != nil {
		t.Fatal(err)
	}
	return hello, resp
}

func TestServesTheContract(t *testing.T) {
	inst := instance(t)
	done := start(t, inst, &clock{t0: time.Now()}, update.UpdateOutput{}, true)

	hello, resp := ask(t, inst, ipc.Request{Op: ipc.OpSnapshot, Key: ipc.Key{SessionDir: "/w"}})
	if hello.Protocol != ipc.Protocol || hello.Version != "v1.0.0" || hello.PID != os.Getpid() {
		t.Errorf("hello = %+v", hello)
	}
	if resp.Snapshot == nil || resp.Snapshot.WorkDir != "/w" {
		t.Errorf("snapshot = %+v", resp)
	}

	_, resp = ask(t, inst, ipc.Request{Op: ipc.OpStatus})
	if st := resp.Status; st == nil || st.Version != "v1.0.0" || st.Sessions != 1 || st.Network.UsageStatus != 429 || st.BadVersion != "v0.0.9" {
		t.Errorf("status = %+v", resp.Status)
	}

	_, resp = ask(t, inst, ipc.Request{Op: "dance"})
	if resp.Error == "" {
		t.Error("an unknown operation was answered without an error")
	}

	info, err := os.Stat(inst.Dir)
	if err != nil || info.Mode().Perm() != 0o700 {
		t.Errorf("instance directory mode = %v, %v", info.Mode().Perm(), err)
	}
	if sock, err := os.Stat(inst.Socket); err != nil || sock.Mode().Perm() != 0o600 {
		t.Errorf("socket mode = %v, %v", sock.Mode().Perm(), err)
	}

	// A second daemon on the same instance leaves at once
	second := daemon.New(daemon.Deps{Clock: &clock{t0: time.Now()}, Registry: state.NewRegistry()})
	if code := second.Run(daemon.RunInput{Instance: inst, Getenv: func(string) string { return "" }}); code != 0 {
		t.Errorf("second daemon exit = %d, want 0", code)
	}

	_, _ = ask(t, inst, ipc.Request{Op: ipc.OpStop})
	select {
	case code := <-done:
		if code != 0 {
			t.Errorf("exit = %d", code)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the daemon did not stop when asked")
	}
	if _, err := os.Stat(inst.Socket); !os.IsNotExist(err) {
		t.Errorf("socket left behind: %v", err)
	}
}

func TestStopsWhenIdle(t *testing.T) {
	inst := instance(t)
	done := start(t, inst, &clock{t0: time.Now(), jump: state.EvictAfter}, update.UpdateOutput{}, false)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("an idle daemon did not stop")
	}
}

func TestStopsAfterAnUpdate(t *testing.T) {
	inst := instance(t)
	done := start(t, inst, &clock{t0: time.Now()}, update.UpdateOutput{Installed: true, Notice: snapshot.UpdateNotice{Available: true, Version: "v1.1.0"}}, false)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the daemon kept running after installing an update")
	}
	if data, err := os.ReadFile(filepath.Clean(inst.Log)); err != nil || len(data) == 0 {
		t.Errorf("no log written: %v", err)
	}
}
