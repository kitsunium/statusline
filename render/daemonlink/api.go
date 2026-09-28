// Package daemonlink is the client's side of statusline.ipc/v1: it asks the
// instance's daemon, and when there is none, or it is cold, older or
// incompatible, renders from the instance's cache and starts a daemon for
// the next redraw. It never stops a newer daemon.
//
// Exported API of design/domains/render.yaml (render/component/daemonlink).
// This file stands in for the shells kit generates (api_gen.go) until
// `kit gen` is available: every exported symbol only delegates to its
// unexported twin.
package daemonlink

import (
	"context"
	"time"

	"github.com/kitsunium/statusline/ipc"
	"github.com/kitsunium/statusline/render/port"
	"github.com/kitsunium/statusline/snapshot"
)

// Config is the client's build and instance.
type Config struct {
	Instance   ipc.Instance
	Version    string
	Executable string
	// Budget bounds one exchange with the daemon; zero means 250 ms.
	Budget time.Duration
}

// Link implements render/port/snapshot-source@v1 and
// render/port/daemon-control@v1.
type Link struct {
	cfg Config
}

var (
	_ port.SnapshotSource = (*Link)(nil)
	_ port.DaemonControl  = (*Link)(nil)
)

// New returns a link to the instance's daemon.
func New(cfg Config) *Link { return newLink(cfg) }

// Snapshot asks the daemon, else reads the cache.
func (l *Link) Snapshot(ctx context.Context, key ipc.Key) (snapshot.Snapshot, port.SnapshotOrigin) {
	return l.snapshot(ctx, key)
}

// Status asks the daemon for its status; an error when none answers.
func (l *Link) Status(ctx context.Context) (ipc.Status, error) { return l.status(ctx) }

// Stop asks the daemon to stop; an error when none answers.
func (l *Link) Stop(ctx context.Context) error { return l.stop(ctx) }
