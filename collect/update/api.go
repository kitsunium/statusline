// Package update keeps the binary up to date silently (D11): the daemon
// looks for a signed release hourly, keeps the previous binary, probes the
// new one and rolls back a bad one for good.
//
// Exported API of design/domains/collect.yaml (collect/component/update).
// This file stands in for the shells kit generates (api_gen.go) until
// `kit gen` is available: every exported symbol only delegates to its
// unexported twin.
package update

import (
	"context"

	"github.com/kitsunium/statusline/collect/port"
	"github.com/kitsunium/statusline/snapshot"
)

// UpdateInput describes the running build.
type UpdateInput struct {
	CurrentVersion string
	// Disabled is set by STATUSLINE_NO_SELF_UPDATE (or the legacy
	// STATUS_LINE_NO_SELF_UPDATE): managed images update themselves.
	Disabled bool
}

// UpdateOutput says what happened.
type UpdateOutput struct {
	Notice    snapshot.UpdateNotice
	Installed bool
}

// CheckUpdate is collect/usecase/check-update.
type CheckUpdate interface {
	Execute(ctx context.Context, in UpdateInput) (UpdateOutput, error)
}

// Deps are the ports the updater links.
type Deps struct {
	Clock    port.Clock
	Store    port.UpdateStore
	Releases port.ReleaseSource
}

// Updater implements CheckUpdate.
type Updater struct {
	deps Deps
}

var _ CheckUpdate = (*Updater)(nil)

// New returns an updater.
func New(deps Deps) *Updater { return &Updater{deps: deps} }

// Execute checks, installs, probes and rolls back when due.
func (u *Updater) Execute(ctx context.Context, in UpdateInput) (UpdateOutput, error) {
	return u.execute(ctx, in)
}
