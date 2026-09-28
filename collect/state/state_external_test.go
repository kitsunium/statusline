package state_test

import (
	"testing"
	"time"

	"pgregory.net/rapid"

	"github.com/kitsunium/statusline/collect/state"
	"github.com/kitsunium/statusline/ipc"
	"github.com/kitsunium/statusline/quota"
	"github.com/kitsunium/statusline/snapshot"
)

var t0 = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func TestRegistryFreshness(t *testing.T) {
	r := state.NewRegistry()
	key := ipc.Key{SessionID: "s"}
	if _, ok := r.Get(key, t0); ok {
		t.Fatal("an empty registry served a snapshot")
	}
	r.Put(key, snapshot.Snapshot{WorkDir: "/w"}, t0)
	if snap, ok := r.Get(key, t0.Add(state.Freshness-time.Millisecond)); !ok || snap.WorkDir != "/w" {
		t.Errorf("a fresh snapshot was not served: %+v %v", snap, ok)
	}
	if _, ok := r.Get(key, t0.Add(state.Freshness)); ok {
		t.Error("a snapshot as old as Freshness was served")
	}
	if _, ok := r.Get(key, t0.Add(-time.Second)); ok {
		t.Error("a snapshot from the future (clock jump) was served")
	}
}

// TestPropertyEvictIdleOnly: Evict removes exactly the keys untouched for
// EvictAfter (collect/property/evict-idle-only).
func TestPropertyEvictIdleOnly(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		r := state.NewRegistry()
		ages := rapid.SliceOfN(rapid.Int64Range(0, int64(3*state.EvictAfter)), 0, 20).Draw(t, "ages")
		now := t0.Add(4 * state.EvictAfter)
		idle := 0
		for i, age := range ages {
			r.Touch(ipc.Key{SessionID: string(rune('a' + i))}, now.Add(-time.Duration(age)))
			if time.Duration(age) >= state.EvictAfter {
				idle++
			}
		}
		if got := r.Evict(now); got != idle {
			t.Fatalf("Evict() = %d, want %d", got, idle)
		}
		if r.Len() != len(ages)-idle {
			t.Fatalf("Len() = %d, want %d", r.Len(), len(ages)-idle)
		}
	})
}

// TestPropertyNeverBeforeRetryAfter: whatever happened, the usage API is
// not due before a recorded Retry-After (collect/property/never-before-retry-after).
func TestPropertyNeverBeforeRetryAfter(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := state.Network{}
		now := t0
		steps := rapid.IntRange(1, 30).Draw(t, "steps")
		for i := 0; i < steps; i++ {
			now = now.Add(time.Duration(rapid.Int64Range(0, int64(2*time.Hour)).Draw(t, "dt")))
			switch rapid.IntRange(0, 2).Draw(t, "event") {
			case 0:
				n = n.RecordUsage(quota.Set{}, now)
			case 1:
				n = n.RecordUsageFailure(429, time.Duration(rapid.Int64Range(0, int64(48*time.Hour)).Draw(t, "retry")), now)
			default:
				n = n.RecordUsageFailure(500, 0, now)
			}
			probe := now.Add(time.Duration(rapid.Int64Range(0, int64(25*time.Hour)).Draw(t, "probe")))
			if probe.Before(n.UsageRetryAfter) && n.UsageDue(probe) {
				t.Fatalf("due at %v before RetryAfter %v", probe, n.UsageRetryAfter)
			}
			if probe.Sub(n.UsageAttemptAt) < state.UsageTTL && n.UsageDue(probe) {
				t.Fatalf("due %v after the last attempt, TTL %v", probe.Sub(n.UsageAttemptAt), state.UsageTTL)
			}
		}
	})
}

// TestPropertyBackoffBounded: after N consecutive failures without a
// Retry-After, the wait is within [UsageTTL, 1 h] and non-decreasing in N
// (collect/property/backoff-bounded).
func TestPropertyBackoffBounded(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		failures := rapid.IntRange(1, 40).Draw(t, "failures")
		n := state.Network{}
		prev := time.Duration(0)
		now := t0
		for i := 0; i < failures; i++ {
			n = n.RecordUsageFailure(500, 0, now)
			wait := n.UsageRetryAfter.Sub(now)
			if wait < state.UsageTTL || wait > time.Hour {
				t.Fatalf("wait %v after %d failures out of [UsageTTL, 1h]", wait, i+1)
			}
			if wait < prev {
				t.Fatalf("wait decreased from %v to %v", prev, wait)
			}
			prev = wait
			now = n.UsageRetryAfter
		}
		n = n.RecordUsage(quota.Set{}, now)
		if n.UsageFailures != 0 || !n.UsageRetryAfter.IsZero() {
			t.Fatalf("a success did not reset the back-off: %+v", n)
		}
	})
}

func TestRetryAfterIsBounded(t *testing.T) {
	n := state.Network{}.RecordUsageFailure(429, time.Second, t0)
	if got := n.UsageRetryAfter.Sub(t0); got != state.UsageTTL {
		t.Errorf("a 1 s Retry-After waits %v, want at least UsageTTL", got)
	}
	n = state.Network{}.RecordUsageFailure(429, 30*24*time.Hour, t0)
	if got := n.UsageRetryAfter.Sub(t0); got != 24*time.Hour {
		t.Errorf("a 30 d Retry-After waits %v, want the 24 h cap", got)
	}
}

// TestHealthAges pins the legacy-only goldens health-aging and health-stale
// (testdata/parity): a summary past its TTL is still drawn, one past
// HealthMaxAge is not.
func TestHealthAges(t *testing.T) {
	n := state.Network{}
	if !n.HealthDue(t0) || n.CurrentHealth(t0) != snapshot.HealthUnknown {
		t.Fatal("no summary yet: due, and unknown")
	}
	n = n.RecordHealth(snapshot.HealthDegraded, t0)
	if n.HealthDue(t0.Add(state.HealthTTL - time.Second)) {
		t.Error("due before HealthTTL")
	}
	aging := t0.Add(5 * time.Minute)
	if !n.HealthDue(aging) || n.CurrentHealth(aging) != snapshot.HealthDegraded {
		t.Error("a 5 min old summary is due for a refresh and still drawn")
	}
	stale := t0.Add(20 * time.Minute)
	if n.CurrentHealth(stale) != snapshot.HealthUnknown {
		t.Error("a 20 min old summary is still drawn")
	}
	failed := n.RecordHealthFailure(aging)
	if failed.CurrentHealth(aging) != snapshot.HealthDegraded || failed.HealthDue(aging) {
		t.Error("a failed fetch must keep the level and wait for the next TTL")
	}
}

func TestUpdateDue(t *testing.T) {
	if !(state.Update{}).Due(t0) {
		t.Error("never checked: due")
	}
	u := state.Update{CheckedAt: t0}
	if u.Due(t0.Add(state.UpdateInterval-time.Second)) || !u.Due(t0.Add(state.UpdateInterval)) {
		t.Error("due exactly every UpdateInterval")
	}
}

func TestRateLimitedCarriesItsCode(t *testing.T) {
	err := &state.RateLimited{RetryAfter: time.Minute}
	if got := err.Error(); got[:len("statusline_rate_limited")] != "statusline_rate_limited" {
		t.Errorf("Error() = %q", got)
	}
}
