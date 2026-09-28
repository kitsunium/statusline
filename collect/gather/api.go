// Package gather collects the snapshot of one session key.
//
// Exported API of design/domains/collect.yaml (collect/component/gather).
// This file stands in for the shells kit generates (api_gen.go) until
// `kit gen` is available: every exported symbol only delegates to its
// unexported twin.
package gather

import (
	"context"

	"github.com/kitsunium/statusline/collect/port"
	"github.com/kitsunium/statusline/collect/refresh"
	"github.com/kitsunium/statusline/collect/state"
	"github.com/kitsunium/statusline/ipc"
	"github.com/kitsunium/statusline/snapshot"
)

// CollectInput names the session key.
type CollectInput struct {
	Key ipc.Key
}

// CollectOutput is the key's snapshot; Collected is false when a fresh one
// was served from the registry.
type CollectOutput struct {
	Snapshot  snapshot.Snapshot
	Collected bool
}

// CollectSnapshot is collect/usecase/collect-snapshot.
type CollectSnapshot interface {
	Execute(ctx context.Context, in CollectInput) (CollectOutput, error)
}

// Deps are the ports and use cases the collector links.
type Deps struct {
	Clock      port.Clock
	WorkDir    port.WorkDir
	Sessions   port.SessionRegistry
	Repository port.Repository
	MCPConfig  port.MCPConfig
	MCPCalls   port.MCPCalls
	Tasks      port.TaskStore
	System     port.SystemInfo
	Cache      port.SnapshotCache
	Network    refresh.LatestNetwork
	Registry   *state.Registry
}

// Collector implements CollectSnapshot.
type Collector struct {
	deps Deps
}

var _ CollectSnapshot = (*Collector)(nil)

// New returns a collector.
func New(deps Deps) *Collector { return &Collector{deps: deps} }

// Execute serves a fresh snapshot or collects one.
func (c *Collector) Execute(ctx context.Context, in CollectInput) (CollectOutput, error) {
	return c.execute(ctx, in)
}
