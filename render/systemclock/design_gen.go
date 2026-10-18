// Code generated from design/v1; DO NOT EDIT.
// design:digest sha256:f7116793c8d8ac76611661ce6107f6662b6eb477309df7bb89556666304dc4e6

// Package systemclock is the adapter component systemclock of domain render.
//
// Its structure is generated from design/ (ADR 0010); its bodies are the
// hand-written twins beside this file.
package systemclock

import (
	"context"
	"time"

	"github.com/kitsunium/sdk/pkg/v1/errs"
	"github.com/kitsunium/statusline/render/show"
)

// errNotImplemented is what a skeleton answers until its body is written:
// a twin that still returns it is a stub.
var errNotImplemented = errs.New(errs.Pack(64, 2, 6, 255), "NOT_IMPLEMENTED", "not implemented yet", "systemclock: a hand-written body is still a skeleton")

// Clock is an adapter of the design.
type Clock struct {
	clock // its own state, hand-written
}

// NewClock builds Clock: its body is the hand-written newClock.
func NewClock() *Clock {
	return newClock()
}

// Now is render/port/Clock@v1's.
func (a *Clock) Now(ctx context.Context) (time.Time, error) {
	return a.now(ctx)
}

var _ show.ClockV1 = (*Clock)(nil)
