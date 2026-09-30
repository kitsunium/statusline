// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package daemon

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kitsunium/sdk/framework/kit"
	sdkipc "github.com/kitsunium/sdk/pkg/v1/ipc"

	"github.com/kitsunium/statusline/collect/collector"
	"github.com/kitsunium/statusline/ipc"
	"github.com/kitsunium/statusline/snapshot"
)

const (
	// connDeadline bounds one exchange; a client that stalls cannot pin a
	// connection.
	connDeadline time.Duration = 5 * time.Second
	// controlBudget bounds a status or stop exchange.
	controlBudget time.Duration = 2 * time.Second
	// filePerm keeps the pid and the heartbeat to their owner.
	filePerm os.FileMode = 0o600
)

// run is the state of this daemon process: the framework runs the listener,
// the jobs and the idle stop; what is left is the product's own.
var run struct {
	once     sync.Once
	instance ipc.Instance
	started  time.Time
	log      *boundedLog
	notice   atomic.Pointer[snapshot.UpdateNotice]
	halting  atomic.Bool
}

// setUp locates the instance and opens its log, once, at the first job or
// connection: the handlers only run in the daemon role.
func setUp(ctx context.Context) {
	run.once.Do(func() {
		run.instance, _ = ipc.Here()
		run.started = now(ctx)
		if run.instance.Dir != "" && privateDir(run.instance.Dir) == nil {
			run.log = openLog(run.instance.Log)
			_ = os.WriteFile(run.instance.PID, []byte(strconv.Itoa(os.Getpid())), filePerm)
		} else {
			run.log = openLog("")
		}
		run.log.printf("started version=%s pid=%d", wired.version, os.Getpid())
	})
}

// now reads the clock; a clock that fails is the wall clock.
func now(ctx context.Context) time.Time {
	if wired.clock == nil {
		return time.Now()
	}
	if t, err := wired.clock.Now(ctx); err == nil {
		return t
	}
	return time.Now()
}

// halt asks the framework to end the daemon's run, which drains every
// component as a signal would.
func halt(ctx context.Context, reason string) {
	if !run.halting.CompareAndSwap(false, true) {
		return
	}
	if run.log != nil {
		run.log.printf("stopping: %s", reason)
	}
	_ = os.Remove(run.instance.PID)
	kit.Stop(ctx)
}

// serve is one connection of the contract: Hello both ways, then the
// operation. The SDK already refused a peer of another account.
func serve(ctx context.Context, conn *sdkipc.Conn) error {
	setUp(ctx)
	_ = conn.SetDeadline(time.Now().Add(connDeadline))
	if err := ipc.WriteFrame(conn, ipc.Hello{Protocol: ipc.Protocol, Version: wired.version, PID: os.Getpid()}); err != nil {
		return nil
	}
	var hello ipc.Hello
	if err := ipc.ReadFrame(conn, &hello); err != nil || !ipc.Compatible(hello.Protocol) {
		return nil
	}
	var req ipc.Request
	if err := ipc.ReadFrame(conn, &req); err != nil {
		return nil
	}
	_ = ipc.WriteFrame(conn, answer(ctx, req))
	if req.Op == ipc.OpStop {
		halt(ctx, "asked by pid "+strconv.Itoa(hello.PID))
	}
	return nil
}

// answer runs one operation of the contract (collect/port/Daemon@v1).
func answer(ctx context.Context, req ipc.Request) ipc.Response {
	switch req.Op {
	case ipc.OpSnapshot:
		out, err := wired.collectSnapshot.Execute(ctx, collector.CollectSnapshotInput{Key: req.Key})
		if err != nil {
			return ipc.Response{Error: err.Error()}
		}
		snap := out.Snapshot
		if notice := run.notice.Load(); notice != nil {
			snap.Update = *notice
		}
		return ipc.Response{Snapshot: &snap}
	case ipc.OpStatus:
		st := daemonStatus(ctx)
		return ipc.Response{Status: &st}
	case ipc.OpStop:
		return ipc.Response{}
	default:
		return ipc.Response{Error: "unknown operation " + req.Op}
	}
}

// daemonStatus reports the daemon, its sessions and its bookkeeping.
func daemonStatus(ctx context.Context) ipc.Status {
	last, _ := wired.sessionCache.LastRequest(ctx)
	sessions, _ := wired.sessionCache.Len(ctx)
	exe, _ := os.Executable()
	st := ipc.Status{
		Version: wired.version, PID: os.Getpid(), Executable: exe,
		StartedAt: run.started, LastRequest: last, Sessions: sessions,
	}
	if latest, err := wired.latestNetwork.Execute(ctx, collector.LatestNetworkInput{}); err == nil {
		n := latest.Network
		st.Network = ipc.Network{
			UsageFetchedAt: n.UsageFetchedAt, UsageAttemptAt: n.UsageAttemptAt,
			UsageRetryAfter: n.UsageRetryAfter, UsageStatus: n.UsageStatus,
			HealthFetchedAt: n.HealthFetchedAt, HealthAttemptAt: n.HealthAttemptAt,
		}
	}
	if u, err := wired.updateStore.LoadUpdate(ctx); err == nil {
		st.BadVersion = u.BadVersion
	}
	return st
}

// tick beats, refreshes what is due and evicts idle sessions. The idle stop
// is the framework's: busy says whether sessions remain.
func tick(ctx context.Context) error {
	setUp(ctx)
	beat()
	if _, err := wired.refreshNetwork.Execute(ctx, collector.RefreshNetworkInput{}); err != nil {
		run.log.printf("refresh: %v", err)
	}
	if n, _ := wired.sessionCache.Evict(ctx, now(ctx)); n > 0 {
		run.log.printf("evicted %d session(s)", n)
	}
	return nil
}

// beat touches the heartbeat: the proof, for a client that got no answer,
// that this daemon still ticks.
func beat() {
	if run.instance.Heartbeat == "" {
		return
	}
	t := time.Now()
	if err := os.Chtimes(run.instance.Heartbeat, t, t); err != nil {
		_ = os.WriteFile(run.instance.Heartbeat, nil, filePerm)
	}
}

// update stops the daemon after an install: the next client finds no daemon
// and starts the new binary.
func update(ctx context.Context) error {
	setUp(ctx)
	in := collector.CheckUpdateInput{CurrentVersion: wired.version, Disabled: updatesDisabled(os.Getenv)}
	out, err := wired.checkUpdate.Execute(ctx, in)
	if out.Installed {
		notice := out.Notice
		run.notice.Store(&notice)
		run.log.printf("installed %s", out.Notice.Version)
		halt(ctx, "updated")
	}
	return err
}

// updatesDisabled honours both spellings of the opt-out.
func updatesDisabled(getenv func(string) string) bool {
	return getenv("STATUSLINE_NO_SELF_UPDATE") != "" || getenv("STATUS_LINE_NO_SELF_UPDATE") != ""
}

// busy keeps the daemon up while a session was asked for within EvictAfter.
func busy(ctx context.Context) bool {
	n, err := wired.sessionCache.Len(ctx)
	return err == nil && n > 0
}

// status runs "statusline daemon status": it asks the instance's daemon
// through the contract, as any client would; no daemon is exit 1, for
// scripts.
func status(ctx context.Context, _ []string, std kit.Stdio) int {
	resp, err := ask(ctx, ipc.OpStatus)
	if err != nil || resp.Status == nil {
		_, _ = fmt.Fprintln(std.Err, "no daemon running")
		return 1
	}
	printStatus(std.Out, *resp.Status)
	return 0
}

// stop runs "statusline daemon stop": stopping no daemon is no error.
func stop(ctx context.Context, _ []string, std kit.Stdio) int {
	if _, err := ask(ctx, ipc.OpStop); err != nil {
		_, _ = fmt.Fprintln(std.Out, "no daemon running")
		return 0
	}
	_, _ = fmt.Fprintln(std.Out, "daemon stopped")
	return 0
}

// ask sends one operation to the instance's daemon.
func ask(ctx context.Context, op string) (ipc.Response, error) {
	inst, err := ipc.Here()
	if err != nil {
		return ipc.Response{}, err
	}
	if _, err := os.Lstat(inst.Socket); err != nil {
		return ipc.Response{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, controlBudget)
	defer cancel()
	conn, err := sdkipc.Dial(ctx, sdkipc.Config{Path: inst.Socket})
	if err != nil {
		return ipc.Response{}, err
	}
	defer func() { _ = conn.Close() }()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	var hello ipc.Hello
	if err := ipc.WriteFrame(conn, ipc.Hello{Protocol: ipc.Protocol, Version: wired.version, PID: os.Getpid()}); err != nil {
		return ipc.Response{}, err
	}
	if err := ipc.ReadFrame(conn, &hello); err != nil {
		return ipc.Response{}, err
	}
	if err := ipc.WriteFrame(conn, ipc.Request{Op: op}); err != nil {
		return ipc.Response{}, err
	}
	var resp ipc.Response
	return resp, ipc.ReadFrame(conn, &resp)
}

// printStatus says what the daemon is doing.
func printStatus(w io.Writer, st ipc.Status) {
	_, _ = fmt.Fprintf(w, "daemon %s pid %d, up %s, %d session(s)\n", st.Version, st.PID, time.Since(st.StartedAt).Round(time.Second), st.Sessions)
	_, _ = fmt.Fprintf(w, "executable %s\n", st.Executable)
	n := st.Network
	_, _ = fmt.Fprintf(w, "usage: fetched %s, attempted %s, status %d, retry after %s\n", stamp(n.UsageFetchedAt), stamp(n.UsageAttemptAt), n.UsageStatus, stamp(n.UsageRetryAfter))
	_, _ = fmt.Fprintf(w, "health: fetched %s, attempted %s\n", stamp(n.HealthFetchedAt), stamp(n.HealthAttemptAt))
	if st.BadVersion != "" {
		_, _ = fmt.Fprintf(w, "bad version (never installed again): %s\n", st.BadVersion)
	}
}

// stamp prints an instant, or "never".
func stamp(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	return t.Local().Format(time.RFC3339)
}
