package daemon

import (
	"context"
	"net"
	"os"
	"time"

	"github.com/kitsunium/statusline/ipc"
)

// connDeadline bounds one exchange; a client that stalls cannot pin a
// goroutine.
const connDeadline time.Duration = 5 * time.Second

// handle serves one request per connection: Hello both ways, then the
// operation. A peer that is not this user is dropped before a byte is read.
func (a *Listener) handle(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	if !samePeer(conn) {
		a.log.printf("refused a peer of another user")
		return
	}
	_ = conn.SetDeadline(time.Now().Add(connDeadline))
	if err := ipc.WriteFrame(conn, ipc.Hello{Protocol: ipc.Protocol, Version: a.version, PID: os.Getpid()}); err != nil {
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
	_ = ipc.WriteFrame(conn, a.answer(req))
	if req.Op == ipc.OpStop {
		_ = a.Stop(context.Background())
	}
}

// answer runs one operation of the contract (collect/port/Daemon@v1).
func (a *Listener) answer(req ipc.Request) ipc.Response {
	ctx := context.Background()
	switch req.Op {
	case ipc.OpSnapshot:
		snap, err := a.Snapshot(ctx, req.Key)
		if err != nil {
			return ipc.Response{Error: err.Error()}
		}
		return ipc.Response{Snapshot: &snap}
	case ipc.OpStatus:
		st, _ := a.Status(ctx)
		return ipc.Response{Status: &st}
	case ipc.OpStop:
		return ipc.Response{}
	default:
		return ipc.Response{Error: "unknown operation " + req.Op}
	}
}
