package daemon

import (
	"context"
	"net"
	"os"
	"time"

	"github.com/kitsunium/statusline/collect/gather"
	"github.com/kitsunium/statusline/collect/refresh"
	"github.com/kitsunium/statusline/ipc"
)

// connDeadline bounds one exchange; a client that stalls cannot pin a
// goroutine.
const connDeadline time.Duration = 5 * time.Second

// handle serves one request per connection: Hello both ways, then the
// operation. A peer that is not this user is dropped before a byte is read.
func (s *server) handle(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	if !samePeer(conn) {
		s.log.printf("refused a peer of another user")
		return
	}
	_ = conn.SetDeadline(time.Now().Add(connDeadline))
	if err := ipc.WriteFrame(conn, ipc.Hello{Protocol: ipc.Protocol, Version: s.in.Version, PID: os.Getpid()}); err != nil {
		return
	}
	var hello ipc.Hello
	if err := ipc.ReadFrame(conn, &hello); err != nil || !ipc.Compatible(hello.Protocol) {
		return
	}
	var req ipc.Request
	if err := ipc.ReadFrame(conn, &req); err != nil {
		return
	}
	resp := s.answer(req)
	_ = ipc.WriteFrame(conn, resp)
	if req.Op == ipc.OpStop {
		s.stop("asked by pid " + itoa(hello.PID))
	}
}

// answer runs one operation.
func (s *server) answer(req ipc.Request) ipc.Response {
	ctx := context.Background()
	switch req.Op {
	case ipc.OpSnapshot:
		out, err := s.deps.Collect.Execute(ctx, gather.CollectInput{Key: req.Key})
		if err != nil {
			return ipc.Response{Error: err.Error()}
		}
		snap := out.Snapshot
		if notice := s.notice.Load(); notice != nil {
			snap.Update = *notice
		}
		return ipc.Response{Snapshot: &snap}
	case ipc.OpStatus:
		st := s.status(ctx)
		return ipc.Response{Status: &st}
	case ipc.OpStop:
		return ipc.Response{}
	default:
		return ipc.Response{Error: "unknown operation " + req.Op}
	}
}

// status reports the daemon, its sessions and its network bookkeeping.
func (s *server) status(ctx context.Context) ipc.Status {
	st := ipc.Status{
		Version:     s.in.Version,
		PID:         os.Getpid(),
		Executable:  s.in.Executable,
		StartedAt:   s.started,
		LastRequest: s.deps.Registry.LastRequest(),
		Sessions:    s.deps.Registry.Len(),
	}
	if latest, err := s.deps.Latest.Execute(ctx, refresh.LatestInput{}); err == nil {
		n := latest.Network
		st.Network = ipc.Network{
			UsageFetchedAt:  n.UsageFetchedAt,
			UsageAttemptAt:  n.UsageAttemptAt,
			UsageRetryAfter: n.UsageRetryAfter,
			UsageStatus:     n.UsageStatus,
			HealthFetchedAt: n.HealthFetchedAt,
			HealthAttemptAt: n.HealthAttemptAt,
		}
	}
	if u, err := s.deps.Updates.LoadUpdate(); err == nil {
		st.BadVersion = u.BadVersion
	}
	return st
}
