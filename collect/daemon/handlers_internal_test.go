package daemon

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kitsunium/sdk/framework/kit"
	sdkipc "github.com/kitsunium/sdk/pkg/v1/ipc"

	"github.com/kitsunium/statusline/collect/collector"
	"github.com/kitsunium/statusline/collect/sessioncache"
	"github.com/kitsunium/statusline/collect/state"
	"github.com/kitsunium/statusline/ipc"
	"github.com/kitsunium/statusline/snapshot"
)

type wallClock struct{}

func (wallClock) Now(context.Context) (time.Time, error) { return time.Now(), nil }

// collect answers the key's directory and touches the cache as the real
// use case does.
type collect struct{ cache collector.SessionCacheV1 }

func (c collect) Execute(ctx context.Context, in collector.CollectSnapshotInput) (collector.CollectSnapshotOutput, error) {
	_ = c.cache.Touch(ctx, in.Key, time.Now())
	return collector.CollectSnapshotOutput{Snapshot: snapshot.Snapshot{WorkDir: in.Key.SessionDir}, Collected: true}, nil
}

type refresh struct{ ticks int }

func (r *refresh) Execute(context.Context, collector.RefreshNetworkInput) (collector.RefreshNetworkOutput, error) {
	r.ticks++
	return collector.RefreshNetworkOutput{}, nil
}

type latest struct{}

func (latest) Execute(context.Context, collector.LatestNetworkInput) (collector.LatestNetworkOutput, error) {
	return collector.LatestNetworkOutput{Network: state.Network{UsageStatus: 429}}, nil
}

type checkUpdate struct{ out collector.CheckUpdateOutput }

func (u checkUpdate) Execute(context.Context, collector.CheckUpdateInput) (collector.CheckUpdateOutput, error) {
	return u.out, nil
}

type updates struct{}

func (updates) LoadUpdate(context.Context) (state.Update, error) {
	return state.Update{BadVersion: "v0.0.9"}, nil
}
func (updates) SaveUpdate(context.Context, state.Update) error { return nil }

// setup wires the handlers on fakes, in a private runtime directory that
// Here computes the instance under.
func setup(t *testing.T) (*sessioncache.Cache, *refresh, ipc.Instance) {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "sl")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	t.Setenv("RUNTIME_DIRECTORY", "")
	t.Setenv("XDG_RUNTIME_DIR", dir)
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(dir, "cfg"))
	cache := sessioncache.NewCache()
	r := &refresh{}
	Wire(collect{cache}, latest{}, cache, updates{}, "v1.0.0", r, wallClock{}, checkUpdate{})
	inst, err := ipc.Here()
	if err != nil {
		t.Fatal(err)
	}
	return cache, r, inst
}

// serveOn listens where the framework's listener would, and serves every
// connection with the handler.
func serveOn(t *testing.T, inst ipc.Instance) {
	t.Helper()
	ln, err := sdkipc.Listen(sdkipc.Config{Path: inst.Socket})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer func() { _ = conn.Close() }()
				_ = Serve(context.Background(), conn)
			}()
		}
	}()
}

func TestServeAndTheCommands(t *testing.T) {
	cache, r, inst := setup(t)
	var out, errb bytes.Buffer
	std := kit.Stdio{Out: &out, Err: &errb}
	if code := Status(context.Background(), nil, std); code != 1 || !strings.Contains(errb.String(), "no daemon") {
		t.Errorf("status without a daemon = %d, %q", code, errb.String())
	}
	out.Reset()
	if code := Stop(context.Background(), nil, std); code != 0 || !strings.Contains(out.String(), "no daemon") {
		t.Errorf("stop without a daemon = %d, %q", code, out.String())
	}

	serveOn(t, inst)
	resp, err := ask(context.Background(), ipc.OpSnapshot)
	if err != nil || resp.Snapshot == nil {
		t.Fatalf("snapshot = %+v, %v", resp, err)
	}
	if err := Tick(context.Background()); err != nil || r.ticks != 1 {
		t.Errorf("Tick() = %v, ticks %d", err, r.ticks)
	}
	if _, err := os.Stat(inst.Heartbeat); err != nil {
		t.Errorf("no heartbeat: %v", err)
	}
	if !Busy(context.Background()) {
		t.Error("a session was asked for, yet the daemon is not busy")
	}
	out.Reset()
	if code := Status(context.Background(), nil, std); code != 0 || !strings.Contains(out.String(), "daemon v1.0.0 pid") || !strings.Contains(out.String(), "v0.0.9") || !strings.Contains(out.String(), "1 session(s)") {
		t.Errorf("status = %d, %q", code, out.String())
	}
	resp, _ = ask(context.Background(), "dance")
	if resp.Error == "" {
		t.Error("an unknown operation was answered without an error")
	}
	_, _ = cache.Evict(context.Background(), time.Now().Add(state.EvictAfter))
	if Busy(context.Background()) {
		t.Error("no session left, yet the daemon is busy")
	}
}

func TestUpdatesDisabled(t *testing.T) {
	env := map[string]string{}
	getenv := func(k string) string { return env[k] }
	if updatesDisabled(getenv) {
		t.Error("disabled by default")
	}
	env["STATUS_LINE_NO_SELF_UPDATE"] = "1"
	if !updatesDisabled(getenv) {
		t.Error("the legacy spelling is not honoured")
	}
}
