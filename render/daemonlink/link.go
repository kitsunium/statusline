package daemonlink

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kitsunium/sdk/pkg/v1/errs"
	sdkipc "github.com/kitsunium/sdk/pkg/v1/ipc"

	"github.com/kitsunium/statusline/ipc"
	"github.com/kitsunium/statusline/render/show"
	"github.com/kitsunium/statusline/snapshot"
)

// defaultBudget bounds one exchange: a warm daemon answers in about a
// millisecond; one collecting a new key takes what the legacy client took.
const defaultBudget time.Duration = 250 * time.Millisecond

// Why a daemon was not used.
var (
	errOlder = errors.New("daemon is older than this client")
	errNewer = errors.New("daemon speaks a newer, incompatible protocol")
	errEmpty = errors.New("daemon answered without a snapshot")
	// errNoSocket says no daemon listens at the instance.
	errNoSocket = errors.New("no daemon socket")
)

// config is what the link knows about the client and its instance.
type config struct {
	Instance   ipc.Instance
	Version    string
	Executable string
	// Budget bounds one exchange with the daemon; zero means defaultBudget.
	Budget time.Duration
}

// link is the Link's own state.
type link struct {
	cfg config
}

// newLink locates the instance of this executable; without one the line
// still renders, from stdin alone.
func newLink(version string) *Link {
	instance, _ := ipc.Here()
	return newLinkWith(config{Instance: instance, Version: version, Executable: executable()})
}

// newLinkWith builds a link on an explicit configuration.
func newLinkWith(cfg config) *Link {
	if cfg.Budget <= 0 {
		cfg.Budget = defaultBudget
	}
	return &Link{link{cfg: cfg}}
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

// snapshot starts a daemon whenever the answer did not come from a
// compatible one of this version or newer; the line is then drawn from the
// key's cache, written by the daemon after each collection.
func (l *Link) snapshot(ctx context.Context, key ipc.Key) (snapshot.Snapshot, string, error) {
	resp, err := l.exchange(ctx, ipc.Request{Op: ipc.OpSnapshot, Key: key}, true)
	if err == nil && resp.Snapshot != nil {
		return *resp.Snapshot, show.OriginDaemon, nil
	}
	// Start one only where none answers, an older one was just stopped, or a
	// mute one was just killed: a slow or newer daemon is left alone
	switch {
	case errors.Is(err, errOlder) || isDialError(err):
		l.start()
	case isTimeout(err) && l.replaceMute():
		l.start()
	}
	if snap, ok := l.readCache(key); ok {
		return snap, show.OriginCache, nil
	}
	return snapshot.Snapshot{}, show.OriginNone, nil
}

// exchange dials, shakes hands and sends one request. With replaceOlder set,
// an older daemon is asked to stop on the same connection instead.
func (l *Link) exchange(ctx context.Context, req ipc.Request, replaceOlder bool) (ipc.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, l.cfg.Budget)
	defer cancel()
	// No socket at all (a first start, a stopped daemon) is nobody to dial:
	// the SDK would read a missing directory as an unsafe one
	if _, err := os.Lstat(l.cfg.Instance.Socket); err != nil {
		return ipc.Response{}, errNoSocket
	}
	conn, err := sdkipc.Dial(ctx, sdkipc.Config{Path: l.cfg.Instance.Socket})
	if err != nil {
		return ipc.Response{}, err
	}
	defer func() { _ = conn.Close() }()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	if err := ipc.WriteFrame(conn, ipc.Hello{Protocol: ipc.Protocol, Version: l.cfg.Version, PID: os.Getpid()}); err != nil {
		return ipc.Response{}, err
	}
	var hello ipc.Hello
	if err := ipc.ReadFrame(conn, &hello); err != nil {
		return ipc.Response{}, err
	}
	older := ipc.CompareVersions(hello.Version, l.cfg.Version) < 0
	switch {
	case older && replaceOlder:
		// The daemon stops after answering; the caller starts this version
		_ = ipc.WriteFrame(conn, ipc.Request{Op: ipc.OpStop})
		var ack ipc.Response
		_ = ipc.ReadFrame(conn, &ack)
		return ipc.Response{}, errOlder
	case !ipc.Compatible(hello.Protocol) && older:
		return ipc.Response{}, errOlder
	case !ipc.Compatible(hello.Protocol):
		return ipc.Response{}, errNewer
	}
	if err := ipc.WriteFrame(conn, req); err != nil {
		return ipc.Response{}, err
	}
	var resp ipc.Response
	if err := ipc.ReadFrame(conn, &resp); err != nil {
		return ipc.Response{}, err
	}
	if req.Op == ipc.OpSnapshot && resp.Snapshot == nil {
		return resp, errEmpty
	}
	return resp, nil
}

// readCache reads the key's last snapshot; a missing or torn file is a
// miss.
func (l *Link) readCache(key ipc.Key) (snapshot.Snapshot, bool) {
	data, err := os.ReadFile(l.cfg.Instance.CachePath(key))
	if err != nil {
		return snapshot.Snapshot{}, false
	}
	var snap snapshot.Snapshot
	if json.Unmarshal(data, &snap) != nil {
		return snapshot.Snapshot{}, false
	}
	return snap, true
}

// start launches `<this executable> daemon`, detached so that it outlives
// this process; a daemon already holding the instance makes it leave at
// once, so a racing start costs one short-lived process.
func (l *Link) start() {
	exe := l.cfg.Executable
	if exe == "" {
		return
	}
	cmd := exec.Command(exe, "daemon")
	cmd.Env = os.Environ()
	cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, nil, nil
	cmd.SysProcAttr = detached()
	if err := cmd.Start(); err != nil {
		return
	}
	_ = cmd.Process.Release()
}

// staleAfter is how long a heartbeat may go untouched: the daemon touches it
// every second, so ten seconds without a beat is a stuck process.
const staleAfter time.Duration = 10 * time.Second

// isTimeout reports an exchange that ran out of its budget.
func isTimeout(err error) bool {
	var ne net.Error
	return errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &ne) && ne.Timeout())
}

// replaceMute kills the instance's daemon when it holds the socket but its
// heartbeat is stale; the kernel then releases its lock. It reports whether
// a daemon was killed.
func (l *Link) replaceMute() bool {
	info, err := os.Stat(l.cfg.Instance.Heartbeat)
	if err != nil || time.Since(info.ModTime()) < staleAfter {
		return false
	}
	data, err := os.ReadFile(l.cfg.Instance.PID)
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 1 || pid == os.Getpid() {
		return false
	}
	return kill(pid) == nil
}

// isDialError reports that no daemon listens: no socket, or a dead one. A
// socket another account owns is refused too, but nothing is started
// beside it.
func isDialError(err error) bool {
	return errors.Is(err, errNoSocket) || errs.HasCode(err, sdkipc.CodeDialFailed)
}
