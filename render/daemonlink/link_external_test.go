package daemonlink_test

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kitsunium/statusline/ipc"
	"github.com/kitsunium/statusline/render/daemonlink"
	"github.com/kitsunium/statusline/render/port"
	"github.com/kitsunium/statusline/snapshot"
)

// fakeDaemon answers the contract with a fixed version and protocol, and
// records the operations it received.
type fakeDaemon struct {
	mu  sync.Mutex
	ops []string
}

func (f *fakeDaemon) serve(t *testing.T, inst ipc.Instance, version, protocol string) {
	t.Helper()
	if err := os.MkdirAll(inst.Dir, 0o700); err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("unix", inst.Socket)
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
				_ = ipc.WriteFrame(conn, ipc.Hello{Protocol: protocol, Version: version, PID: 1})
				var hello ipc.Hello
				if ipc.ReadFrame(conn, &hello) != nil {
					return
				}
				var req ipc.Request
				if ipc.ReadFrame(conn, &req) != nil {
					return
				}
				f.mu.Lock()
				f.ops = append(f.ops, req.Op)
				f.mu.Unlock()
				switch req.Op {
				case ipc.OpSnapshot:
					_ = ipc.WriteFrame(conn, ipc.Response{Snapshot: &snapshot.Snapshot{WorkDir: "from-daemon"}})
				case ipc.OpStatus:
					_ = ipc.WriteFrame(conn, ipc.Response{Status: &ipc.Status{Version: version}})
				default:
					_ = ipc.WriteFrame(conn, ipc.Response{})
				}
			}()
		}
	}()
}

func (f *fakeDaemon) seen() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return strings.Join(f.ops, ",")
}

// setup returns an instance under a short path, and an executable that
// records being started as a daemon.
func setup(t *testing.T) (ipc.Instance, string, string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("a shell script stands for the binary")
	}
	dir, err := os.MkdirTemp("/tmp", "sl")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	marker := filepath.Join(dir, "started")
	exe := filepath.Join(dir, "statusline")
	script := "#!/bin/sh\necho \"$@\" >> " + marker + "\n"
	if err := os.WriteFile(exe, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	inst := ipc.Locate(ipc.LocateInput{RuntimeDir: dir, UID: os.Getuid(), ConfigDir: "/c", Executable: exe})
	return inst, exe, marker
}

// started waits a moment for the detached start and reports it.
func started(marker string) bool {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if data, err := os.ReadFile(marker); err == nil && strings.TrimSpace(string(data)) == "daemon" {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

func TestWarmDaemon(t *testing.T) {
	inst, exe, marker := setup(t)
	var d fakeDaemon
	d.serve(t, inst, "v1.0.0", ipc.Protocol)
	link := daemonlink.New(daemonlink.Config{Instance: inst, Version: "v1.0.0", Executable: exe})
	snap, origin := link.Snapshot(context.Background(), ipc.Key{SessionID: "s"})
	if origin != port.OriginDaemon || snap.WorkDir != "from-daemon" {
		t.Errorf("Snapshot() = %+v, %v", snap, origin)
	}
	if st, err := link.Status(context.Background()); err != nil || st.Version != "v1.0.0" {
		t.Errorf("Status() = %+v, %v", st, err)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Error("a daemon was started while one answered")
	}
}

// TestSequenceRenderCold pins design/sequences/render-cold.yaml: no daemon,
// the cache is read and a daemon is started.
func TestSequenceRenderCold(t *testing.T) {
	inst, exe, marker := setup(t)
	key := ipc.Key{SessionID: "s", SessionDir: "/w"}
	link := daemonlink.New(daemonlink.Config{Instance: inst, Version: "v1.0.0", Executable: exe})

	start := time.Now()
	snap, origin := link.Snapshot(context.Background(), key)
	if origin != port.OriginNone || snap.WorkDir != "" {
		t.Errorf("nothing cached: %+v, %v", snap, origin)
	}
	if !started(marker) {
		t.Error("no daemon was started")
	}

	if err := os.MkdirAll(inst.Cache, 0o700); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(snapshot.Snapshot{WorkDir: "from-cache"})
	if err := os.WriteFile(inst.CachePath(key), data, 0o600); err != nil {
		t.Fatal(err)
	}
	snap, origin = link.Snapshot(context.Background(), key)
	if origin != port.OriginCache || snap.WorkDir != "from-cache" {
		t.Errorf("cached: %+v, %v", snap, origin)
	}
	if took := time.Since(start); took > 2*time.Second {
		t.Errorf("cold renders took %v", took)
	}
}

func TestOlderDaemonIsReplaced(t *testing.T) {
	inst, exe, marker := setup(t)
	var d fakeDaemon
	d.serve(t, inst, "v0.9.0", ipc.Protocol)
	link := daemonlink.New(daemonlink.Config{Instance: inst, Version: "v1.0.0", Executable: exe})
	if _, origin := link.Snapshot(context.Background(), ipc.Key{}); origin == port.OriginDaemon {
		t.Error("an older daemon's snapshot was used")
	}
	if got := d.seen(); got != ipc.OpStop {
		t.Errorf("older daemon received %q, want stop", got)
	}
	if !started(marker) {
		t.Error("this version's daemon was not started")
	}
}

func TestNewerDaemonIsNeverStopped(t *testing.T) {
	inst, exe, marker := setup(t)
	var d fakeDaemon
	d.serve(t, inst, "v2.0.0", "statusline.ipc/v2")
	link := daemonlink.New(daemonlink.Config{Instance: inst, Version: "v1.0.0", Executable: exe})
	if _, origin := link.Snapshot(context.Background(), ipc.Key{}); origin == port.OriginDaemon {
		t.Error("an incompatible daemon's answer was used")
	}
	if got := d.seen(); got != "" {
		t.Errorf("newer daemon received %q, want nothing", got)
	}
	time.Sleep(200 * time.Millisecond)
	if _, err := os.Stat(marker); err == nil {
		t.Error("a daemon was started beside a newer one")
	}
}

func TestNewerCompatibleDaemonIsUsed(t *testing.T) {
	inst, exe, _ := setup(t)
	var d fakeDaemon
	d.serve(t, inst, "v1.4.0", ipc.Protocol)
	link := daemonlink.New(daemonlink.Config{Instance: inst, Version: "v1.0.0", Executable: exe})
	if _, origin := link.Snapshot(context.Background(), ipc.Key{}); origin != port.OriginDaemon {
		t.Errorf("origin = %v, want the newer compatible daemon", origin)
	}
}

func TestControlWithoutDaemon(t *testing.T) {
	inst, exe, marker := setup(t)
	link := daemonlink.New(daemonlink.Config{Instance: inst, Version: "v1.0.0", Executable: exe})
	if _, err := link.Status(context.Background()); err == nil {
		t.Error("Status() without a daemon did not fail")
	}
	if err := link.Stop(context.Background()); err == nil {
		t.Error("Stop() without a daemon did not fail")
	}
	time.Sleep(200 * time.Millisecond)
	if _, err := os.Stat(marker); err == nil {
		t.Error("status or stop started a daemon")
	}
}
