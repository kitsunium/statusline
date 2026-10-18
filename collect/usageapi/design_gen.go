// Code generated from design/v1; DO NOT EDIT.
// design:digest sha256:f7116793c8d8ac76611661ce6107f6662b6eb477309df7bb89556666304dc4e6

// Package usageapi is the adapter component usageapi of domain collect.
//
// Its structure is generated from design/ (ADR 0010); its bodies are the
// hand-written twins beside this file.
package usageapi

import (
	"context"

	"github.com/kitsunium/sdk/pkg/v1/errs"
	"github.com/kitsunium/statusline/collect/collector"
	"github.com/kitsunium/statusline/quota"
)

// errNotImplemented is what a skeleton answers until its body is written:
// a twin that still returns it is a stub.
var errNotImplemented = errs.New(errs.Pack(64, 3, 10, 255), "NOT_IMPLEMENTED", "not implemented yet", "usageapi: a hand-written body is still a skeleton")

// Usage is an adapter of the design.
type Usage struct {
	usage // its own state, hand-written
}

// NewUsage builds Usage: its body is the hand-written newUsage.
func NewUsage() *Usage {
	return newUsage()
}

// Fetch is collect/port/UsageAPI@v1's.
func (a *Usage) Fetch(ctx context.Context, token string) (quota.Set, error) {
	return a.fetch(ctx, token)
}

var _ collector.UsageAPIV1 = (*Usage)(nil)
