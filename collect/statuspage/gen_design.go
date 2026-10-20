// Code generated from design/v1; DO NOT EDIT.
// design:digest sha256:f7116793c8d8ac76611661ce6107f6662b6eb477309df7bb89556666304dc4e6

// Package statuspage is the adapter component statuspage of domain collect.
//
// Its structure is generated from design/ (ADR 0010); its bodies are the
// hand-written twins beside this file.
package statuspage

import (
	"context"

	"github.com/kitsunium/sdk/pkg/v1/errs"
	"github.com/kitsunium/statusline/collect/collector"
	"github.com/kitsunium/statusline/snapshot"
)

// errNotImplemented is what a skeleton answers until its body is written:
// a twin that still returns it is a stub.
var errNotImplemented = errs.New(errs.Pack(64, 3, 12, 255), "NOT_IMPLEMENTED", "not implemented yet", "statuspage: a hand-written body is still a skeleton")

// StatusPage is an adapter of the design.
type StatusPage struct {
	statusPage // its own state, hand-written
}

// NewStatusPage builds StatusPage: its body is the hand-written newStatusPage.
func NewStatusPage() *StatusPage {
	return newStatusPage()
}

// Fetch is collect/port/StatusPage@v1's.
func (a *StatusPage) Fetch(ctx context.Context) (snapshot.Health, error) {
	return a.fetch(ctx)
}

var _ collector.StatusPageV1 = (*StatusPage)(nil)
