// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package sessioncache

import (
	"context"
	"sync"
	"time"

	"github.com/kitsunium/statusline/collect/state"
	"github.com/kitsunium/statusline/ipc"
	"github.com/kitsunium/statusline/snapshot"
)

// cache holds session states by key (D5 — session, transcript, directory);
// safe for concurrent use.
type cache struct {
	mu      sync.Mutex
	entries map[ipc.Key]*entry
	last    time.Time
}

// entry is one key's state.
type entry struct {
	snap        snapshot.Snapshot
	collectedAt time.Time
	requestedAt time.Time
	has         bool
}

func newCache() *Cache {
	return &Cache{cache{entries: make(map[ipc.Key]*entry)}}
}

// evict drops exactly the keys whose last request is state.EvictAfter old.
func (a *Cache) evict(_ context.Context, now time.Time) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	n := 0
	for key, e := range a.entries {
		if now.Sub(e.requestedAt) >= state.EvictAfter {
			delete(a.entries, key)
			n++
		}
	}
	return n, nil
}

// get serves a snapshot younger than state.Freshness; one from the future
// (a clock jump) is not served either.
func (a *Cache) get(_ context.Context, key ipc.Key, now time.Time) (snapshot.Snapshot, bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	e, ok := a.entries[key]
	if !ok || !e.has || now.Sub(e.collectedAt) >= state.Freshness || now.Before(e.collectedAt) {
		return snapshot.Snapshot{}, false, nil
	}
	return e.snap, true, nil
}

func (a *Cache) lastRequest(context.Context) (time.Time, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.last, nil
}

func (a *Cache) lenBody(context.Context) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.entries), nil
}

func (a *Cache) put(_ context.Context, key ipc.Key, snap snapshot.Snapshot, now time.Time) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	e := a.entry(key)
	e.snap, e.collectedAt, e.has = snap, now, true
	return nil
}

func (a *Cache) touch(_ context.Context, key ipc.Key, now time.Time) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.entry(key).requestedAt = now
	if now.After(a.last) {
		a.last = now
	}
	return nil
}

// entry returns the key's entry, created on first use; the caller holds mu.
func (a *Cache) entry(key ipc.Key) *entry {
	e, ok := a.entries[key]
	if !ok {
		e = &entry{}
		a.entries[key] = e
	}
	return e
}
