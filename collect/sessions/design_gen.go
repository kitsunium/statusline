// Code generated from design/v1; DO NOT EDIT.
// design:digest sha256:f7116793c8d8ac76611661ce6107f6662b6eb477309df7bb89556666304dc4e6

// Package sessions is the adapter component sessions of domain collect.
//
// Its structure is generated from design/ (ADR 0010); its bodies are the
// hand-written twins beside this file.
package sessions

import (
	"context"

	"github.com/kitsunium/sdk/pkg/v1/errs"
	"github.com/kitsunium/statusline/collect/collector"
	"github.com/kitsunium/statusline/collect/state"
)

// errNotImplemented is what a skeleton answers until its body is written:
// a twin that still returns it is a stub.
var errNotImplemented = errs.New(errs.Pack(64, 3, 7, 255), "NOT_IMPLEMENTED", "not implemented yet", "sessions: a hand-written body is still a skeleton")

// Sessions is an adapter of the design.
type Sessions struct {
	sessions // its own state, hand-written
}

// NewSessions builds Sessions: its body is the hand-written newSessions.
func NewSessions() *Sessions {
	return newSessions()
}

// Lookup is collect/port/SessionRegistry@v1's.
func (a *Sessions) Lookup(ctx context.Context, sessionID string) (state.HostSession, bool, error) {
	return a.lookup(ctx, sessionID)
}

var _ collector.SessionRegistryV1 = (*Sessions)(nil)
