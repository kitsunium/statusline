package taskstore

import (
	"context"
	"time"

	"github.com/kitsunium/statusline/snapshot"
)

// tasks holds nothing: every call reads the stores afresh.
type tasks struct{}

func newTasks() *Tasks { return &Tasks{} }

// board reads one session at one instant.
func (a *Tasks) board(_ context.Context, sessionID, listID string, now time.Time) (snapshot.TaskBoard, error) {
	return newBoard(sessionID, listID, now).board(), nil
}
