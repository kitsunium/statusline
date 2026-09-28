// Package daemon is the daemon role's entry: one instance per (UID, host
// configuration directory, executable) under a lock, a private socket that
// only accepts its own user, the operations of statusline.ipc/v1, the loops,
// and the stop once no session asked for a minute.
//
// Exported API of design/domains/collect.yaml (collect/component/daemon).
// This file stands in for the shells kit generates (api_gen.go) until
// `kit gen` is available: every exported symbol only delegates to its
// unexported twin.
package daemon

import (
	"github.com/kitsunium/statusline/collect/gather"
	"github.com/kitsunium/statusline/collect/port"
	"github.com/kitsunium/statusline/collect/refresh"
	"github.com/kitsunium/statusline/collect/state"
	"github.com/kitsunium/statusline/collect/update"
	"github.com/kitsunium/statusline/ipc"
)

// Deps are the use cases the daemon serves.
type Deps struct {
	Clock    port.Clock
	Registry *state.Registry
	Collect  gather.CollectSnapshot
	Refresh  refresh.RefreshNetwork
	Latest   refresh.LatestNetwork
	Update   update.CheckUpdate
	Updates  port.UpdateStore
}

// RunInput is the daemon's invocation.
type RunInput struct {
	Version    string
	Executable string
	Instance   ipc.Instance
	Getenv     func(string) string
}

// Daemon serves the client role.
type Daemon struct {
	deps Deps
}

// New returns a daemon.
func New(deps Deps) *Daemon { return &Daemon{deps: deps} }

// Run serves until the daemon has nothing left to do; it returns the exit
// code (0 also when another daemon already holds the instance).
func (d *Daemon) Run(in RunInput) int { return d.run(in) }
