// Package show renders the status line of one redraw: the host's payload,
// the daemon's snapshot of the session, and the stdin quotas winning.
//
// Exported API of design/domains/render.yaml (render/component/show).
// This file stands in for the shells kit generates (api_gen.go) until
// `kit gen` is available: every exported symbol only delegates to its
// unexported twin.
package show

import (
	"context"

	"github.com/kitsunium/statusline/render/port"
)

// ShowInput is one redraw.
type ShowInput struct {
	// Stdin is the host's payload, as read.
	Stdin []byte
	// Columns is COLUMNS, the room the host gives the line.
	Columns string
	// TaskListID is CLAUDE_CODE_TASK_LIST_ID.
	TaskListID string
	// Getenv reads the display switches.
	Getenv func(string) string
}

// ShowOutput is the line to print and where its snapshot came from.
type ShowOutput struct {
	Line   string
	Origin port.SnapshotOrigin
}

// ShowStatusLine is render/usecase/show-status-line.
type ShowStatusLine interface {
	Execute(ctx context.Context, in ShowInput) (ShowOutput, error)
}

// Deps are the ports the use case links.
type Deps struct {
	Clock     port.Clock
	Snapshots port.SnapshotSource
}

// Shower implements ShowStatusLine.
type Shower struct {
	deps Deps
}

var _ ShowStatusLine = (*Shower)(nil)

// New returns the use case.
func New(deps Deps) *Shower { return &Shower{deps: deps} }

// Execute renders one redraw; it never fails on a bad payload.
func (s *Shower) Execute(ctx context.Context, in ShowInput) (ShowOutput, error) {
	return s.execute(ctx, in)
}
