// Code generated from design/v1; DO NOT EDIT.
// design:digest sha256:f7116793c8d8ac76611661ce6107f6662b6eb477309df7bb89556666304dc4e6

// Package show — One redraw.
//
// Its structure is generated from design/ (ADR 0010); its bodies are the
// hand-written twins beside this file.
package show

import (
	"context"
	"time"

	"github.com/kitsunium/sdk/pkg/v1/errs"
	"github.com/kitsunium/statusline/ipc"
	"github.com/kitsunium/statusline/snapshot"
)

// errNotImplemented is what a skeleton answers until its body is written:
// a twin that still returns it is a stub.
var errNotImplemented = errs.New(errs.Pack(64, 2, 3, 255), "NOT_IMPLEMENTED", "not implemented yet", "show: a hand-written body is still a skeleton")

const (
	// OriginCache is a constant of the design.
	OriginCache string = "cache"
	// OriginDaemon is a constant of the design.
	OriginDaemon string = "daemon"
	// OriginNone is a constant of the design.
	OriginNone string = "none"
)

// ClockV1 is an outbound port: the product calls through it.
type ClockV1 interface {
	Now(ctx context.Context) (time.Time, error)
}

// SnapshotSourceV1 gives a session's snapshot and where it came from
// (Origin*).
type SnapshotSourceV1 interface {
	Snapshot(ctx context.Context, key ipc.Key) (snapshot.Snapshot, string, error)
}

// ShowStatusLine — The host's payload keyed as the daemon keeps it, the
// snapshot drawn at the clock's instant, stdin's quotas winning.
type ShowStatusLine interface {
	Execute(ctx context.Context, in ShowStatusLineInput) (ShowStatusLineOutput, error)
}

// ShowStatusLineInput is ShowStatusLine's input: a field added breaks no caller.
type ShowStatusLineInput struct {
	Stdin []byte
	Env   map[string]string
}

// ShowStatusLineOutput is ShowStatusLine's output.
type ShowStatusLineOutput struct {
	Line   string
	Origin string
}

// ShowStatusLineHandler runs ShowStatusLine: its body is the hand-written execute.
type ShowStatusLineHandler struct {
	snapshotSource SnapshotSourceV1
	clock          ClockV1
}

// NewShowStatusLine builds ShowStatusLine on what it calls.
func NewShowStatusLine(snapshotSource SnapshotSourceV1, clock ClockV1) *ShowStatusLineHandler {
	return &ShowStatusLineHandler{snapshotSource: snapshotSource, clock: clock}
}

// Execute runs ShowStatusLine.
func (h *ShowStatusLineHandler) Execute(ctx context.Context, in ShowStatusLineInput) (ShowStatusLineOutput, error) {
	return h.execute(ctx, &in)
}

var _ ShowStatusLine = (*ShowStatusLineHandler)(nil)
