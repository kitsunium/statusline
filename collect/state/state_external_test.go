package state_test

import (
	"testing"
	"time"

	"github.com/kitsunium/statusline/collect/state"
	"github.com/kitsunium/statusline/snapshot"
)

var t0 = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

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
