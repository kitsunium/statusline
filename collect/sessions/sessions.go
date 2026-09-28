package sessions

import (
	"context"

	"github.com/kitsunium/statusline/collect/state"
)

// sessions is where the host's session registry lives.
type sessions struct {
	dir string
}

func newSessions() *Sessions { return &Sessions{sessions{dir: registryDir()}} }

func (a *Sessions) lookup(_ context.Context, sessionID string) (state.HostSession, bool, error) {
	h, ok := lookup(a.dir, sessionID)
	return h, ok, nil
}
