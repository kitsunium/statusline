// Code generated from design/v1; DO NOT EDIT.
// design:digest sha256:f7116793c8d8ac76611661ce6107f6662b6eb477309df7bb89556666304dc4e6

// Package daemonlink — The client's side of the contract. Dials under a 250
// ms budget; no daemon, an older one (asked to stop) or a mute one (stale
// heartbeat, killed) → the key's cache and a daemon started with
// os.Executable(). Never stops a newer daemon.
//
// Its structure is generated from design/ (ADR 0010); its bodies are the
// hand-written twins beside this file.
package daemonlink

import (
	"context"

	"github.com/kitsunium/sdk/pkg/v1/errs"
	"github.com/kitsunium/statusline/ipc"
	"github.com/kitsunium/statusline/render/show"
	"github.com/kitsunium/statusline/snapshot"
)

// errNotImplemented is what a skeleton answers until its body is written:
// a twin that still returns it is a stub.
var errNotImplemented = errs.New(errs.Pack(64, 2, 5, 255), "NOT_IMPLEMENTED", "not implemented yet", "daemonlink: a hand-written body is still a skeleton")

// Link is an adapter of the design.
type Link struct {
	link // its own state, hand-written
}

// NewLink builds Link: its body is the hand-written newLink.
func NewLink(version string) *Link {
	return newLink(version)
}

// Snapshot is render/port/SnapshotSource@v1's.
func (a *Link) Snapshot(ctx context.Context, key ipc.Key) (snapshot.Snapshot, string, error) {
	return a.snapshot(ctx, key)
}

var _ show.SnapshotSourceV1 = (*Link)(nil)
