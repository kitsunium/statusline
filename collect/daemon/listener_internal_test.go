package daemon

import (
	"bytes"
	"context"
	"errors"
	"net"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kitsunium/statusline/collect/collector"
	"github.com/kitsunium/statusline/collect/sessioncache"
	"github.com/kitsunium/statusline/collect/state"
	"github.com/kitsunium/statusline/ipc"
	"github.com/kitsunium/statusline/snapshot"
)

// clock returns t0, then jumps by jump on every later call.
type clock struct {
	t0    time.Time
	jump  time.Duration
	calls atomic.Int64
}

func (c *clock) Now(context.Context) (time.Time, error) {
	n := c.calls.Add(1) - 1
	return c.t0.Add(time.Duration(n) * c.jump), nil
}

// collect answers the key's directory and touches the cache as the real
// use case does.
type collect struct{ cache collector.SessionCacheV1 }

func (c collect) Execute(ctx context.Context, in collector.CollectSnapshotInput) (collector.CollectSnapshotOutput, error) {
	_ = c.cache.Touch(ctx, in.Key, time.Now())
	return collector.CollectSnapshotOutput{Snapshot: snapshot.Snapshot{WorkDir: in.Key.SessionDir}, Collected: true}, nil
}

type refresh struct{}

func (refresh) Execute(context.Context, collector.RefreshNetworkInput) (collector.RefreshNetworkOutput, error) {
	return collector.RefreshNetworkOutput{}, nil
}

type latest struct{}

func (latest) Execute(context.Context, collector.LatestNetworkInput) (collector.LatestNetworkOutput, error) {
	return collector.LatestNetworkOutput{Network: state.Network{UsageStatus: 429}}, nil
}

type update struct{ out collector.CheckUpdateOutput }

func (u update) Execute(context.Context, collector.CheckUpdateInput) (collector.CheckUpdateOutput, error) {
	return u.out, nil
}

type updates struct{}

func (updates) LoadUpdate(context.Context) (state.Update, error) {
	return state.Update{BadVersion: "v0.0.9"}, nil
}
func (updates) SaveUpdate(context.Context, state.Update) error { return nil }

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

// listenerAt builds a daemon on fakes at an explicit instance.
func listenerAt(inst ipc.Instance, clk *clock, up collector.CheckUpdateOutput) *Listener {
	cache := sessioncache.NewCache()
	l := newListener(collect{cache}, latest{}, refresh{}, update{up}, cache, updates{}, clk, "v1.0.0")
	l.instance, l.version, l.executable = inst, "v1.0.0", "/e"
	l.getenv = func(string) string { return "" }
	return l
}

func start(t *testing.T, l *Listener, wait bool) <-chan error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- l.Run(context.Background(), nil) }()
	if !wait {
		return done
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(l.instance.Socket); err == nil {
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
	done := start(t, listenerAt(inst, &clock{t0: time.Now()}, collector.CheckUpdateOutput{}), true)

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
	if _, err := os.Stat(inst.PID); err != nil {
		t.Errorf("no pid file: %v", err)
	}
	if _, err := os.Stat(inst.Heartbeat); err != nil {
		t.Errorf("no heartbeat: %v", err)
	}

	// A second daemon on the same instance leaves at once, successfully
	if err := listenerAt(inst, &clock{t0: time.Now()}, collector.CheckUpdateOutput{}).Run(context.Background(), nil); err != nil {
		t.Errorf("second daemon = %v, want nil", err)
	}

	_, _ = ask(t, inst, ipc.Request{Op: ipc.OpStop})
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run() = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the daemon did not stop when asked")
	}
	if _, err := os.Stat(inst.Socket); !os.IsNotExist(err) {
		t.Errorf("socket left behind: %v", err)
	}
}

func TestStopsWhenIdle(t *testing.T) {
	done := start(t, listenerAt(instance(t), &clock{t0: time.Now(), jump: state.EvictAfter}, collector.CheckUpdateOutput{}), false)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("an idle daemon did not stop")
	}
}

func TestStopsAfterAnUpdate(t *testing.T) {
	inst := instance(t)
	up := collector.CheckUpdateOutput{Installed: true, Notice: snapshot.UpdateNotice{Available: true, Version: "v1.1.0"}}
	done := start(t, listenerAt(inst, &clock{t0: time.Now()}, up), false)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the daemon kept running after installing an update")
	}
	if data, err := os.ReadFile(inst.Log); err != nil || len(data) == 0 {
		t.Errorf("no log written: %v", err)
	}
}

func TestStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	l := listenerAt(instance(t), &clock{t0: time.Now()}, collector.CheckUpdateOutput{})
	done := make(chan error, 1)
	go func() { done <- l.Run(ctx, nil) }()
	time.Sleep(200 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("a signal did not stop the daemon")
	}
}

func TestDaemonStatusAndStopCommands(t *testing.T) {
	inst := instance(t)
	ctl := listenerAt(inst, &clock{t0: time.Now()}, collector.CheckUpdateOutput{})
	var out bytes.Buffer
	ctl.stdout = &out
	if err := ctl.Run(context.Background(), []string{"status"}); !errors.Is(err, errNoDaemon) {
		t.Errorf("status without a daemon = %q, %v; want an error for scripts", out.String(), err)
	}
	out.Reset()
	if err := ctl.Run(context.Background(), []string{"stop"}); err != nil || !strings.Contains(out.String(), "no daemon") {
		t.Errorf("stop without a daemon = %q, %v", out.String(), err)
	}
	if err := ctl.Run(context.Background(), []string{"dance"}); err == nil {
		t.Error("an unknown command was accepted")
	}

	done := start(t, listenerAt(inst, &clock{t0: time.Now()}, collector.CheckUpdateOutput{}), true)
	out.Reset()
	if err := ctl.Run(context.Background(), []string{"status"}); err != nil || !strings.Contains(out.String(), "daemon v1.0.0 pid") || !strings.Contains(out.String(), "v0.0.9") {
		t.Errorf("status = %q, %v", out.String(), err)
	}
	out.Reset()
	if err := ctl.Run(context.Background(), []string{"stop"}); err != nil || !strings.Contains(out.String(), "daemon stopped") {
		t.Errorf("stop = %q, %v", out.String(), err)
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("daemon stop did not stop the daemon")
	}
}
