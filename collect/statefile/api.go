// Package statefile persists the daemon's state under its instance
// directory: the network bookkeeping, the update bookkeeping, and the last
// snapshot of every key for the clients to render from when the daemon is
// cold.
//
// Exported API of design/domains/collect.yaml (collect/component/statefile).
// This file stands in for the shells kit generates (api_gen.go) until
// `kit gen` is available: every exported symbol only delegates to its
// unexported twin.
package statefile

import (
	"github.com/kitsunium/statusline/collect/state"
	"github.com/kitsunium/statusline/ipc"
	"github.com/kitsunium/statusline/snapshot"
)

// Store implements collect/port/network-store@v1, collect/port/snapshot-cache@v1
// and collect/port/update-store@v1.
type Store struct {
	instance ipc.Instance
}

// New returns the store of an instance.
func New(instance ipc.Instance) *Store { return &Store{instance: instance} }

// LoadNetwork reads the network bookkeeping; the zero value when none.
func (s *Store) LoadNetwork() (state.Network, error) { return s.loadNetwork() }

// SaveNetwork writes the network bookkeeping atomically.
func (s *Store) SaveNetwork(n state.Network) error { return s.saveNetwork(n) }

// SaveSnapshot writes a key's last snapshot atomically.
func (s *Store) SaveSnapshot(key ipc.Key, snap snapshot.Snapshot) error {
	return s.saveSnapshot(key, snap)
}

// LoadUpdate reads the update bookkeeping; the zero value when none.
func (s *Store) LoadUpdate() (state.Update, error) { return s.loadUpdate() }

// SaveUpdate writes the update bookkeeping atomically.
func (s *Store) SaveUpdate(u state.Update) error { return s.saveUpdate(u) }
