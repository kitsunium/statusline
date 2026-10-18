// Code generated from design/v1; DO NOT EDIT.
// design:digest sha256:f7116793c8d8ac76611661ce6107f6662b6eb477309df7bb89556666304dc4e6

// Package taskstore is the adapter component taskstore of domain collect.
//
// Its structure is generated from design/ (ADR 0010); its bodies are the
// hand-written twins beside this file.
package taskstore

import (
	"context"
	"time"

	"github.com/kitsunium/sdk/pkg/v1/errs"
	"github.com/kitsunium/statusline/collect/collector"
	"github.com/kitsunium/statusline/snapshot"
)

// errNotImplemented is what a skeleton answers until its body is written:
// a twin that still returns it is a stub.
var errNotImplemented = errs.New(errs.Pack(64, 3, 8, 255), "NOT_IMPLEMENTED", "not implemented yet", "taskstore: a hand-written body is still a skeleton")

// Tasks is an adapter of the design.
type Tasks struct {
	tasks // its own state, hand-written
}

// NewTasks builds Tasks: its body is the hand-written newTasks.
func NewTasks() *Tasks {
	return newTasks()
}

// Board is collect/port/TaskStore@v1's.
func (a *Tasks) Board(ctx context.Context, sessionID string, listID string, now time.Time) (snapshot.TaskBoard, error) {
	return a.board(ctx, sessionID, listID, now)
}

var _ collector.TaskStoreV1 = (*Tasks)(nil)
