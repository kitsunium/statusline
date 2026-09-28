package daemon

import (
	"context"
	"errors"
	"net"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kitsunium/sdk/pkg/v1/lock"

	"github.com/kitsunium/statusline/collect/refresh"
	"github.com/kitsunium/statusline/collect/state"
	"github.com/kitsunium/statusline/collect/update"
	"github.com/kitsunium/statusline/snapshot"
)

const (
	// tickEvery paces the refresh, eviction and update loop.
	tickEvery time.Duration = time.Second
	// lockName is the instance lock inside the instance directory.
	lockName string = "daemon"
	// socketPerm keeps the socket to its owner, on top of the 0700 directory.
	socketPerm os.FileMode = 0o600
)

// server is one running daemon.
type server struct {
	deps     Deps
	in       RunInput
	started  time.Time
	log      *boundedLog
	stopOnce sync.Once
	stopped  chan struct{}
	updating atomic.Bool
	notice   atomic.Pointer[snapshot.UpdateNotice]
	conns    sync.WaitGroup
}

// run holds the instance lock for its whole life: a second daemon started
// by a racing client finds it taken and leaves at once.
func (d *Daemon) run(in RunInput) int {
	if err := privateDir(in.Instance.Dir); err != nil {
		return 1
	}
	log := openLog(in.Instance.Log)
	defer log.close()
	locker, err := lock.NewFileLocker(lock.FileConfig{Dir: in.Instance.Dir})
	if err != nil {
		log.printf("lock: %v", err)
		return 1
	}
	ctx := context.Background()
	lease, held, err := locker.TryAcquire(ctx, lockName)
	if err != nil {
		log.printf("lock: %v", err)
		return 1
	}
	if !held {
		return 0
	}
	defer func() { _ = lease.Release(ctx) }()

	// The lock is ours, so any socket left there is a dead daemon's
	_ = os.Remove(in.Instance.Socket)
	ln, err := net.Listen("unix", in.Instance.Socket)
	if err != nil {
		log.printf("listen: %v", err)
		return 1
	}
	defer func() { _ = os.Remove(in.Instance.Socket) }()
	if err := os.Chmod(in.Instance.Socket, socketPerm); err != nil {
		_ = ln.Close()
		log.printf("chmod socket: %v", err)
		return 1
	}

	_ = os.WriteFile(in.Instance.PID, []byte(strconv.Itoa(os.Getpid())), socketPerm)
	defer func() { _ = os.Remove(in.Instance.PID) }()
	s := &server{deps: d.deps, in: in, started: d.deps.Clock.Now(), log: log, stopped: make(chan struct{})}
	log.printf("started version=%s pid=%d", in.Version, os.Getpid())
	go s.accept(ln)
	s.loop()
	_ = ln.Close()
	s.conns.Wait()
	log.printf("stopped")
	return 0
}

// stop ends the daemon once; the reason goes to the log.
func (s *server) stop(reason string) {
	s.stopOnce.Do(func() {
		s.log.printf("stopping: %s", reason)
		close(s.stopped)
	})
}

// loop ticks until stopped: first at once, so that a new daemon has its
// network figures before the second redraw.
func (s *server) loop() {
	ticker := time.NewTicker(tickEvery)
	defer ticker.Stop()
	for {
		s.tick()
		select {
		case <-s.stopped:
			return
		case <-ticker.C:
		}
	}
}

// tick refreshes what is due, evicts idle sessions, stops when none is left
// for EvictAfter, and starts an update check in the background.
func (s *server) tick() {
	s.beat()
	ctx := context.Background()
	if _, err := s.deps.Refresh.Execute(ctx, refresh.RefreshInput{}); err != nil {
		s.log.printf("refresh: %v", err)
	}
	now := s.deps.Clock.Now()
	if n := s.deps.Registry.Evict(now); n > 0 {
		s.log.printf("evicted %d session(s)", n)
	}
	idleSince := s.started
	if last := s.deps.Registry.LastRequest(); last.After(idleSince) {
		idleSince = last
	}
	if s.deps.Registry.Len() == 0 && now.Sub(idleSince) >= state.EvictAfter {
		s.stop("idle")
		return
	}
	if s.updating.CompareAndSwap(false, true) {
		go s.checkUpdate()
	}
}

// beat touches the heartbeat: the proof, for a client that got no answer,
// that this daemon still ticks.
func (s *server) beat() {
	now := time.Now()
	if err := os.Chtimes(s.in.Instance.Heartbeat, now, now); err != nil {
		_ = os.WriteFile(s.in.Instance.Heartbeat, nil, socketPerm)
	}
}

// checkUpdate stops the daemon after an install: the next client finds no
// daemon and starts the new binary.
func (s *server) checkUpdate() {
	defer s.updating.Store(false)
	in := update.UpdateInput{CurrentVersion: s.in.Version, Disabled: updatesDisabled(s.in.Getenv)}
	out, err := s.deps.Update.Execute(context.Background(), in)
	if err != nil {
		s.log.printf("update: %v", err)
	}
	if out.Installed {
		notice := out.Notice
		s.notice.Store(&notice)
		s.log.printf("installed %s", out.Notice.Version)
		s.stop("updated")
	}
}

// updatesDisabled honours both spellings of the opt-out.
func updatesDisabled(getenv func(string) string) bool {
	return getenv("STATUSLINE_NO_SELF_UPDATE") != "" || getenv("STATUS_LINE_NO_SELF_UPDATE") != ""
}

// accept serves connections until the listener closes.
func (s *server) accept(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			s.log.printf("accept: %v", err)
			continue
		}
		s.conns.Add(1)
		go func() {
			defer s.conns.Done()
			s.handle(conn)
		}()
	}
}
