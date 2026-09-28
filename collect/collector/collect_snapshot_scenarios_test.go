// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package collector

import (
	"testing"

	"github.com/kitsunium/statusline/collect/state"
	"github.com/kitsunium/statusline/ipc"
	"github.com/kitsunium/statusline/snapshot"
)

// collecting builds CollectSnapshot on fakes, the network figures included.
func collecting(src *sources, cache *memCache, clk *clock) CollectSnapshot {
	net := &network{n: stateWithScoped()}
	latest := NewLatestNetwork(clk, net)
	return NewCollectSnapshot(clk, cache, src, src, src, src, src, src, src, latest, cache)
}

var sessionKey = ipc.Key{SessionID: "s", TranscriptPath: "/t.jsonl", SessionDir: "/work/session"}

func givenCollectSnapshotCollectsAndCachesANewKey(t *testing.T) (CollectSnapshot, CollectSnapshotInput, func(*testing.T, CollectSnapshotOutput)) {
	t.Helper()
	src, cache := &sources{}, newMemCache()
	return collecting(src, cache, &clock{now: t0}), CollectSnapshotInput{Key: sessionKey}, func(t *testing.T, out CollectSnapshotOutput) {
		snap := out.Snapshot
		if !out.Collected || snap.WorkDir != "/work/elsewhere" || src.gitDir != "/work/elsewhere" {
			t.Errorf("git must run where the session works: %+v, git in %q", out, src.gitDir)
		}
		if !snap.Working || len(snap.MCP) != 1 || !snap.MCP[0].Busy || snap.System.OS != snapshot.OSDarwin {
			t.Errorf("snapshot = %+v", snap)
		}
		if snap.Health != snapshot.HealthDegraded || len(snap.API.Scoped) != 1 {
			t.Errorf("network figures missing: %+v", snap)
		}
		if len(cache.saved) != 1 || cache.saved[0] != sessionKey || cache.touchd != 1 {
			t.Errorf("cached %v, touched %d", cache.saved, cache.touchd)
		}
	}
}

func givenCollectSnapshotServesAFreshSnapshotWithoutReadingAnything(t *testing.T) (CollectSnapshot, CollectSnapshotInput, func(*testing.T, CollectSnapshotOutput)) {
	t.Helper()
	src, cache, clk := &sources{}, newMemCache(), &clock{now: t0}
	uc := collecting(src, cache, clk)
	if _, err := uc.Execute(t.Context(), CollectSnapshotInput{Key: sessionKey}); err != nil {
		t.Fatal(err)
	}
	before := src.total()
	clk.now = t0.Add(state.Freshness / 2)
	return uc, CollectSnapshotInput{Key: sessionKey}, func(t *testing.T, out CollectSnapshotOutput) {
		if out.Collected || src.total() != before || out.Snapshot.WorkDir != "/work/elsewhere" {
			t.Errorf("a fresh snapshot was collected again: %+v, %d calls", out, src.total()-before)
		}
	}
}
