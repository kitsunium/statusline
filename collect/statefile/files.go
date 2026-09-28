package statefile

import (
	"context"
	"sync"

	"github.com/kitsunium/statusline/collect/state"
	"github.com/kitsunium/statusline/ipc"
	"github.com/kitsunium/statusline/snapshot"
)

// files is the instance whose state it keeps, and the network bookkeeping
// held in memory once read: the refresh reads it every second.
type files struct {
	instance ipc.Instance
	mu       sync.Mutex
	network  *state.Network
}

// newFiles locates the instance of this executable.
func newFiles() *Files {
	instance, _ := ipc.Here()
	return newFilesAt(instance)
}

// newFilesAt keeps the state of an explicit instance.
func newFilesAt(instance ipc.Instance) *Files { return &Files{files{instance: instance}} }

// loadNetwork reads the file once, then serves the copy in memory.
func (a *Files) loadNetwork(context.Context) (state.Network, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.network != nil {
		return *a.network, nil
	}
	n, err := loadNetwork(a.instance)
	if err != nil {
		return state.Network{}, err
	}
	a.network = &n
	return n, nil
}

// saveNetwork writes through: memory and file.
func (a *Files) saveNetwork(_ context.Context, n state.Network) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.network = &n
	return saveNetwork(a.instance, n)
}

func (a *Files) saveSnapshot(_ context.Context, key ipc.Key, snap snapshot.Snapshot) error {
	return saveSnapshot(a.instance, key, snap)
}

func (a *Files) loadUpdate(context.Context) (state.Update, error) { return loadUpdate(a.instance) }

func (a *Files) saveUpdate(_ context.Context, u state.Update) error { return saveUpdate(a.instance, u) }
