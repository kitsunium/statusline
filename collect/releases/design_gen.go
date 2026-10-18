// Code generated from design/v1; DO NOT EDIT.
// design:digest sha256:f7116793c8d8ac76611661ce6107f6662b6eb477309df7bb89556666304dc4e6

// Package releases — The SDK's signed self-update: vendor keys, signature
// domain, automatic consent, no elevation, <bin>.prev with a probe and a
// rollback.
//
// Its structure is generated from design/ (ADR 0010); its bodies are the
// hand-written twins beside this file.
package releases

import (
	"context"

	"github.com/kitsunium/sdk/pkg/v1/errs"
	"github.com/kitsunium/statusline/collect/collector"
	"github.com/kitsunium/statusline/collect/state"
)

// errNotImplemented is what a skeleton answers until its body is written:
// a twin that still returns it is a stub.
var errNotImplemented = errs.New(errs.Pack(64, 3, 15, 255), "NOT_IMPLEMENTED", "not implemented yet", "releases: a hand-written body is still a skeleton")

// Releases is an adapter of the design.
type Releases struct {
	releases // its own state, hand-written
}

// NewReleases builds Releases: its body is the hand-written newReleases.
func NewReleases(version string, vendorKey string) *Releases {
	return newReleases(version, vendorKey)
}

// Install is collect/port/ReleaseSource@v1's.
func (a *Releases) Install(ctx context.Context, rel state.Release) error {
	return a.install(ctx, rel)
}

// Latest is collect/port/ReleaseSource@v1's.
func (a *Releases) Latest(ctx context.Context) (state.Release, error) {
	return a.latest(ctx)
}

var _ collector.ReleaseSourceV1 = (*Releases)(nil)
