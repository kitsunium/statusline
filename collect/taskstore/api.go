// Package taskstore reads a session's task board: the tasks MCP file of
// kodflow-hooks when it exists, else the host's native task list, and the
// running subagents.
//
// Exported API of design/domains/collect.yaml (collect/component/taskstore).
// This file stands in for the shells kit generates (api_gen.go) until
// `kit gen` is available: every exported symbol only delegates to its
// unexported twin.
package taskstore

import (
	"time"

	"github.com/kitsunium/statusline/snapshot"
)

// Store implements collect/port/task-store@v1.
type Store struct{}

// New returns a task store reader.
func New() *Store { return &Store{} }

// Board returns the main agent's open epics in display order, each with its
// subagents, and the count of subagents tied to none shown.
func (s *Store) Board(sessionID, listID string, now time.Time) snapshot.TaskBoard {
	return s.board(sessionID, listID, now)
}
