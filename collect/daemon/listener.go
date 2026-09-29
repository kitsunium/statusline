// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package daemon

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kitsunium/sdk/pkg/v1/errs"
	sdkipc "github.com/kitsunium/sdk/pkg/v1/ipc"
	"github.com/kitsunium/sdk/pkg/v1/lock"

	"github.com/kitsunium/statusline/collect/collector"
	"github.com/kitsunium/statusline/collect/state"
	"github.com/kitsunium/statusline/ipc"
	"github.com/kitsunium/statusline/snapshot"
)

const (
	// tickEvery paces the refresh, eviction, heartbeat and update loop.
	tickEvery time.Duration = time.Second
	// lockName is the instance lock inside the instance directory.
	lockName string = "daemon"
	// filePerm keeps the socket, pid and heartbeat to their owner.
	filePerm os.FileMode = 0o600
)

// listener is the daemon: its use cases and ports, what it knows of itself,
// and the state of the run.
type listener struct {
	collect  collector.CollectSnapshot
	latest   collector.LatestNetwork
	refresh  collector.RefreshNetwork
	update   collector.CheckUpdate
	sessions collector.SessionCacheV1
	updates  collector.UpdateStoreV1
	clock    collector.ClockV1

	instance   ipc.Instance
	version    string
	executable string
	getenv     func(string) string
	stdout     io.Writer

	started  time.Time
	log      *boundedLog
	stopOnce sync.Once
	stopped  chan struct{}
	updating atomic.Bool
	notice   atomic.Pointer[snapshot.UpdateNotice]
	conns    sync.WaitGroup
}

// newListener locates the instance of this executable.
func newListener(collectSnapshot collector.CollectSnapshot, latestNetwork collector.LatestNetwork, refreshNetwork collector.RefreshNetwork, checkUpdate collector.CheckUpdate, sessionCache collector.SessionCacheV1, updateStore collector.UpdateStoreV1, clock collector.ClockV1, version string) *Listener {
	instance, _ := ipc.Here()
	return &Listener{listener{
		collect: collectSnapshot, latest: latestNetwork, refresh: refreshNetwork, update: checkUpdate,
		sessions: sessionCache, updates: updateStore, clock: clock,
		instance: instance, version: version, executable: executable(), getenv: os.Getenv,
		stopped: make(chan struct{}),
	}}
}

// run holds the instance lock for its whole life: a second daemon started
// by a racing client finds it taken and leaves at once, successfully.
func (a *Listener) run(ctx context.Context, args []string) error {
	if len(args) > 0 {
		return a.control(ctx, args)
	}
	if a.instance.Dir == "" {
		return errors.New("statusline daemon: no instance directory")
	}
	if err := privateDir(a.instance.Dir); err != nil {
		return err
	}
	a.log = openLog(a.instance.Log)
	defer a.log.close()
	locker, err := lock.NewFileLocker(lock.FileConfig{Dir: a.instance.Dir})
	if err != nil {
		return err
	}
	lease, held, err := locker.TryAcquire(ctx, lockName)
	if err != nil {
		return err
	}
	if !held {
		return nil
	}
	defer func() { _ = lease.Release(context.Background()) }()

	// The SDK's private socket: 0600 in the 0700 instance directory, a dead
	// daemon's socket replaced, a live one refused, and on Linux a peer of
	// another account closed before Accept returns it (SO_PEERCRED)
	ln, err := sdkipc.Listen(sdkipc.Config{Path: a.instance.Socket})
	if err != nil {
		return err
	}
	_ = os.WriteFile(a.instance.PID, []byte(strconv.Itoa(os.Getpid())), filePerm)
	defer func() { _ = os.Remove(a.instance.PID) }()

	a.started = a.now(ctx)
	a.log.printf("started version=%s pid=%d", a.version, os.Getpid())
	go a.accept(ln)
	go func() {
		select {
		case <-ctx.Done():
			a.halt("signal")
		case <-a.stopped:
		}
	}()
	a.loop()
	_ = ln.Close()
	a.conns.Wait()
	a.log.printf("stopped")
	return nil
}

// snapshot is the contract's snapshot operation, with the update notice.
func (a *Listener) snapshot(ctx context.Context, key ipc.Key) (snapshot.Snapshot, error) {
	out, err := a.collect.Execute(ctx, collector.CollectSnapshotInput{Key: key})
	if err != nil {
		return snapshot.Snapshot{}, err
	}
	snap := out.Snapshot
	if notice := a.notice.Load(); notice != nil {
		snap.Update = *notice
	}
	return snap, nil
}

// status reports the daemon, its sessions and its network bookkeeping.
func (a *Listener) status(ctx context.Context) (ipc.Status, error) {
	last, _ := a.sessions.LastRequest(ctx)
	sessions, _ := a.sessions.Len(ctx)
	st := ipc.Status{
		Version: a.version, PID: os.Getpid(), Executable: a.executable,
		StartedAt: a.started, LastRequest: last, Sessions: sessions,
	}
	if latest, err := a.latest.Execute(ctx, collector.LatestNetworkInput{}); err == nil {
		n := latest.Network
		st.Network = ipc.Network{
			UsageFetchedAt: n.UsageFetchedAt, UsageAttemptAt: n.UsageAttemptAt,
			UsageRetryAfter: n.UsageRetryAfter, UsageStatus: n.UsageStatus,
			HealthFetchedAt: n.HealthFetchedAt, HealthAttemptAt: n.HealthAttemptAt,
		}
	}
	if u, err := a.updates.LoadUpdate(ctx); err == nil {
		st.BadVersion = u.BadVersion
	}
	return st, nil
}

// stop is the contract's stop: the daemon ends after answering.
func (a *Listener) stop(context.Context) error {
	a.halt("asked")
	return nil
}

// halt ends the daemon once; the reason goes to the log.
func (a *Listener) halt(reason string) {
	a.stopOnce.Do(func() {
		a.log.printf("stopping: %s", reason)
		close(a.stopped)
	})
}

// now reads the clock; a clock that fails is the wall clock.
func (a *Listener) now(ctx context.Context) time.Time {
	t, err := a.clock.Now(ctx)
	if err != nil {
		return time.Now()
	}
	return t
}

// loop ticks until stopped: first at once, so that a new daemon has its
// network figures before the second redraw.
func (a *Listener) loop() {
	ticker := time.NewTicker(tickEvery)
	defer ticker.Stop()
	for {
		a.tick()
		select {
		case <-a.stopped:
			return
		case <-ticker.C:
		}
	}
}

// tick beats, refreshes what is due, evicts idle sessions, stops when none
// is left for EvictAfter, and starts an update check in the background.
func (a *Listener) tick() {
	a.beat()
	ctx := context.Background()
	if _, err := a.refresh.Execute(ctx, collector.RefreshNetworkInput{}); err != nil {
		a.log.printf("refresh: %v", err)
	}
	now := a.now(ctx)
	if n, _ := a.sessions.Evict(ctx, now); n > 0 {
		a.log.printf("evicted %d session(s)", n)
	}
	idleSince := a.started
	if last, _ := a.sessions.LastRequest(ctx); last.After(idleSince) {
		idleSince = last
	}
	if left, _ := a.sessions.Len(ctx); left == 0 && now.Sub(idleSince) >= state.EvictAfter {
		a.halt("idle")
		return
	}
	if a.updating.CompareAndSwap(false, true) {
		go a.checkUpdate()
	}
}

// beat touches the heartbeat: the proof, for a client that got no answer,
// that this daemon still ticks.
func (a *Listener) beat() {
	now := time.Now()
	if err := os.Chtimes(a.instance.Heartbeat, now, now); err != nil {
		_ = os.WriteFile(a.instance.Heartbeat, nil, filePerm)
	}
}

// checkUpdate stops the daemon after an install: the next client finds no
// daemon and starts the new binary.
func (a *Listener) checkUpdate() {
	defer a.updating.Store(false)
	in := collector.CheckUpdateInput{CurrentVersion: a.version, Disabled: updatesDisabled(a.getenv)}
	out, err := a.update.Execute(context.Background(), in)
	if err != nil {
		a.log.printf("update: %v", err)
	}
	if out.Installed {
		notice := out.Notice
		a.notice.Store(&notice)
		a.log.printf("installed %s", out.Notice.Version)
		a.halt("updated")
	}
}

// updatesDisabled honours both spellings of the opt-out.
func updatesDisabled(getenv func(string) string) bool {
	return getenv("STATUSLINE_NO_SELF_UPDATE") != "" || getenv("STATUS_LINE_NO_SELF_UPDATE") != ""
}

// accept serves the admitted connections until the listener closes.
func (a *Listener) accept(ln *sdkipc.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			if errs.HasCode(err, sdkipc.CodeClosed) {
				return
			}
			a.log.printf("accept: %v", err)
			continue
		}
		a.conns.Add(1)
		go func() {
			defer a.conns.Done()
			a.handle(conn)
		}()
	}
}

// executable is this binary, symbolic links resolved.
func executable() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		return resolved
	}
	return exe
}
