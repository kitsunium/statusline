// Code generated from design/v1; DO NOT EDIT.
// design:digest sha256:f7116793c8d8ac76611661ce6107f6662b6eb477309df7bb89556666304dc4e6

// Package sysinfo is the adapter component sysinfo of domain collect.
//
// Its structure is generated from design/ (ADR 0010); its bodies are the
// hand-written twins beside this file.
package sysinfo

import (
	"context"

	"github.com/kitsunium/sdk/pkg/v1/errs"
	"github.com/kitsunium/statusline/collect/collector"
	"github.com/kitsunium/statusline/snapshot"
)

// errNotImplemented is what a skeleton answers until its body is written:
// a twin that still returns it is a stub.
var errNotImplemented = errs.New(errs.Pack(64, 3, 9, 255), "NOT_IMPLEMENTED", "not implemented yet", "sysinfo: a hand-written body is still a skeleton")

// System is an adapter of the design.
type System struct {
	system // its own state, hand-written
}

// NewSystem builds System: its body is the hand-written newSystem.
func NewSystem() *System {
	return newSystem()
}

// Info is collect/port/SystemInfo@v1's.
func (a *System) Info(ctx context.Context) (snapshot.System, error) {
	return a.info(ctx)
}

var _ collector.SystemInfoV1 = (*System)(nil)
