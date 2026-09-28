// Package ctl holds the user's commands about the daemon: status and stop.
//
// Exported API of design/domains/render.yaml (render/component/ctl). This
// file stands in for the shells kit generates (api_gen.go) until `kit gen`
// is available: every exported symbol only delegates to its unexported twin.
package ctl

import (
	"context"

	"github.com/kitsunium/statusline/ipc"
	"github.com/kitsunium/statusline/render/port"
)

// StatusInput asks for the daemon's status.
type StatusInput struct{}

// StatusOutput is the daemon's status; Running is false without a daemon.
type StatusOutput struct {
	Running bool
	Status  ipc.Status
}

// DaemonStatus is render/usecase/daemon-status.
type DaemonStatus interface {
	Execute(ctx context.Context, in StatusInput) (StatusOutput, error)
}

// StopInput asks the daemon to stop.
type StopInput struct{}

// StopOutput says whether a daemon was running.
type StopOutput struct {
	WasRunning bool
}

// StopDaemon is render/usecase/stop-daemon.
type StopDaemon interface {
	Execute(ctx context.Context, in StopInput) (StopOutput, error)
}

// Deps are the ports the commands link.
type Deps struct {
	Daemon port.DaemonControl
}

// Status implements DaemonStatus.
type Status struct{ deps Deps }

// Stop implements StopDaemon.
type Stop struct{ deps Deps }

var (
	_ DaemonStatus = (*Status)(nil)
	_ StopDaemon   = (*Stop)(nil)
)

// NewStatus returns the status command.
func NewStatus(deps Deps) *Status { return &Status{deps: deps} }

// NewStop returns the stop command.
func NewStop(deps Deps) *Stop { return &Stop{deps: deps} }

// Execute asks the daemon for its status.
func (c *Status) Execute(ctx context.Context, in StatusInput) (StatusOutput, error) {
	return c.execute(ctx, in)
}

// Execute asks the daemon to stop.
func (c *Stop) Execute(ctx context.Context, in StopInput) (StopOutput, error) {
	return c.execute(ctx, in)
}
