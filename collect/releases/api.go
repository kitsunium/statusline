// Package releases finds, installs, probes and rolls back releases of the
// binary through the SDK's signed self-update.
//
// Exported API of design/domains/collect.yaml (collect/component/releases).
// This file stands in for the shells kit generates (api_gen.go) until
// `kit gen` is available: every exported symbol only delegates to its
// unexported twin.
package releases

import (
	"context"

	"github.com/kitsunium/statusline/collect/state"
)

// Config is what a build knows about itself.
type Config struct {
	// Version is the running release; empty or "dev" for a development build.
	Version string
	// Executable is the path of the running binary.
	Executable string
	// VendorKey is the ed25519 key releases are signed with; without it
	// nothing is ever installed.
	VendorKey []byte
}

// Source implements collect/port/release-source@v1.
type Source struct {
	cfg Config
	svc updater
}

// New returns a release source for the kitsunium/statusline releases.
func New(cfg Config) *Source { return newSource(cfg) }

// Latest returns the newest release; the zero Release when none is newer.
func (s *Source) Latest(ctx context.Context) (state.Release, error) { return s.latest(ctx) }

// Install keeps <bin>.prev then replaces the binary with the release.
func (s *Source) Install(ctx context.Context, rel state.Release) error { return s.install(ctx, rel) }

// Probe runs the installed binary with --version and checks its answer.
func (s *Source) Probe(ctx context.Context) error { return s.probe(ctx) }

// Rollback puts <bin>.prev back.
func (s *Source) Rollback(ctx context.Context) error { return s.rollback(ctx) }
