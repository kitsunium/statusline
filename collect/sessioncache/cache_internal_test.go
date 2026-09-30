package sessioncache

import (
	"context"
	"testing"
	"time"

	"pgregory.net/rapid"

	"github.com/kitsunium/statusline/collect/state"
	"github.com/kitsunium/statusline/ipc"
	"github.com/kitsunium/statusline/snapshot"
)

var (
	t0  = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx = context.Background()
)

func TestRegistryFreshness(t *testing.T) {
	r := newCache()
	key := ipc.Key{SessionID: "s"}
	if _, ok, _ := r.Get(ctx, key, t0); ok {
		t.Fatal("an empty registry served a snapshot")
	}
	_ = r.Put(ctx, key, snapshot.Snapshot{WorkDir: "/w"}, t0)
	if snap, ok, _ := r.Get(ctx, key, t0.Add(state.Freshness-time.Millisecond)); !ok || snap.WorkDir != "/w" {
		t.Errorf("a fresh snapshot was not served: %+v %v", snap, ok)
	}
	if _, ok, _ := r.Get(ctx, key, t0.Add(state.Freshness)); ok {
		t.Error("a snapshot as old as Freshness was served")
	}
	if _, ok, _ := r.Get(ctx, key, t0.Add(-time.Second)); ok {
		t.Error("a snapshot from the future (clock jump) was served")
	}
}

// TestPropertyEvictIdleOnly: Evict removes exactly the keys untouched for
// EvictAfter (collect/property/evict-idle-only).
func TestPropertyEvictIdleOnly(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		r := newCache()
		ages := rapid.SliceOfN(rapid.Int64Range(0, int64(3*state.EvictAfter)), 0, 20).Draw(t, "ages")
		now := t0.Add(4 * state.EvictAfter)
		idle := 0
		for i, age := range ages {
			_ = r.Touch(ctx, ipc.Key{SessionID: string(rune('a' + i))}, now.Add(-time.Duration(age)))
			if time.Duration(age) >= state.EvictAfter {
				idle++
			}
		}
		if got, _ := r.Evict(ctx, now); got != idle {
			t.Fatalf("Evict() = %d, want %d", got, idle)
		}
		if n, _ := r.Len(ctx); n != len(ages)-idle {
			t.Fatalf("Len() = %d, want %d", n, len(ages)-idle)
		}
	})
}

func TestLastRequest(t *testing.T) {
	r := newCache()
	_ = r.Touch(ctx, ipc.Key{SessionID: "a"}, t0.Add(time.Minute))
	_ = r.Touch(ctx, ipc.Key{SessionID: "b"}, t0)
	if last, _ := r.LastRequest(ctx); !last.Equal(t0.Add(time.Minute)) {
		t.Errorf("LastRequest() = %v, want the latest touch", last)
	}
}
