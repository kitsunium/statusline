// Package refresh keeps the network figures fresh — the usage API at most
// once a minute, the status page every two — without ever hammering an
// endpoint, and serves the last ones it got.
//
// Exported API of design/domains/collect.yaml (collect/component/refresh).
// This file stands in for the shells kit generates (api_gen.go) until
// `kit gen` is available: every exported symbol only delegates to its
// unexported twin.
package refresh

import (
	"context"
	"sync"

	"github.com/kitsunium/statusline/collect/port"
	"github.com/kitsunium/statusline/collect/state"
	"github.com/kitsunium/statusline/quota"
	"github.com/kitsunium/statusline/snapshot"
)

// RefreshInput asks for a refresh tick.
type RefreshInput struct {
	// Force fetches whatever is not held back by a Retry-After.
	Force bool
}

// RefreshOutput says what was fetched.
type RefreshOutput struct {
	UsageFetched  bool
	HealthFetched bool
}

// RefreshNetwork is collect/usecase/refresh-network.
type RefreshNetwork interface {
	Execute(ctx context.Context, in RefreshInput) (RefreshOutput, error)
}

// LatestInput asks for the last figures.
type LatestInput struct{}

// LatestOutput is the last figures.
type LatestOutput struct {
	Usage    quota.Set
	HasUsage bool
	Health   snapshot.Health
	Network  state.Network
}

// LatestNetwork is collect/usecase/latest-network.
type LatestNetwork interface {
	Execute(ctx context.Context, in LatestInput) (LatestOutput, error)
}

// Deps are the ports the refresher links (design: links of the use cases).
type Deps struct {
	Clock  port.Clock
	Store  port.NetworkStore
	Tokens port.TokenStore
	Usage  port.UsageAPI
	Status port.StatusPage
}

// Refresher implements RefreshNetwork; Latest() implements LatestNetwork.
type Refresher struct {
	deps   Deps
	mu     sync.Mutex
	net    state.Network
	loaded bool
}

var (
	_ RefreshNetwork = (*Refresher)(nil)
	_ LatestNetwork  = latest{}
)

// New returns a refresher.
func New(deps Deps) *Refresher { return &Refresher{deps: deps} }

// Execute runs one refresh tick.
func (r *Refresher) Execute(ctx context.Context, in RefreshInput) (RefreshOutput, error) {
	return r.execute(ctx, in)
}

// Latest returns the LatestNetwork use case of this refresher.
func (r *Refresher) Latest() LatestNetwork { return latest{r: r} }

// latest is LatestNetwork over a refresher's state.
type latest struct{ r *Refresher }

// Execute returns the last figures.
func (l latest) Execute(ctx context.Context, in LatestInput) (LatestOutput, error) {
	return l.r.latest(ctx, in)
}
