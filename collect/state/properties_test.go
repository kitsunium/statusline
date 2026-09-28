// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package state

import (
	"testing"
	"time"

	"pgregory.net/rapid"

	"github.com/kitsunium/statusline/quota"
)

var t0 = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// propertyNetworkUsageDueNeverBeforeRetryAfter: whatever happened, the usage API is
// not due before a recorded Retry-After (collect/property/never-before-retry-after).
func propertyNetworkUsageDueNeverBeforeRetryAfter(t *testing.T, seed uint64) {
	t.Logf("design seed %d (rapid v1.2 takes its own seed from -rapid.seed)", seed)
	rapid.Check(t, func(t *rapid.T) {
		n := Network{}
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
			if probe.Sub(n.UsageAttemptAt) < UsageTTL && n.UsageDue(probe) {
				t.Fatalf("due %v after the last attempt, TTL %v", probe.Sub(n.UsageAttemptAt), UsageTTL)
			}
		}
	})
}

// propertyNetworkRecordUsageFailureBackoffBounded: after N consecutive failures without a
// Retry-After, the wait is within [UsageTTL, 1 h] and non-decreasing in N
// (collect/property/backoff-bounded).
func propertyNetworkRecordUsageFailureBackoffBounded(t *testing.T, seed uint64) {
	t.Logf("design seed %d (rapid v1.2 takes its own seed from -rapid.seed)", seed)
	rapid.Check(t, func(t *rapid.T) {
		failures := rapid.IntRange(1, 40).Draw(t, "failures")
		n := Network{}
		prev := time.Duration(0)
		now := t0
		for i := 0; i < failures; i++ {
			n = n.RecordUsageFailure(500, 0, now)
			wait := n.UsageRetryAfter.Sub(now)
			if wait < UsageTTL || wait > time.Hour {
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

func seeded(t *testing.T, seed uint64) {
	t.Helper()
	t.Logf("design seed %d (rapid v1.2 takes its own seed from -rapid.seed)", seed)
}

func instantGen() *rapid.Generator[time.Time] {
	return rapid.Custom(func(t *rapid.T) time.Time {
		return t0.Add(time.Duration(rapid.Int64Range(0, int64(48*time.Hour)).Draw(t, "at")))
	})
}

func propertyNetworkCurrentHealthUnknownPastMaxAge(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		at, now := instantGen().Draw(t, "at"), instantGen().Draw(t, "now")
		n := Network{}.RecordHealth(rapid.IntRange(1, 3).Draw(t, "h"), at)
		got := n.CurrentHealth(now)
		if fresh := now.Sub(at) < HealthMaxAge; fresh != (got == n.Health) || (!fresh && got != 0) {
			t.Fatalf("CurrentHealth at %v after = %d", now.Sub(at), got)
		}
	})
}

func propertyNetworkHealthDueTTL(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		at, now := instantGen().Draw(t, "at"), instantGen().Draw(t, "now")
		n := Network{}.RecordHealthFailure(at)
		if n.HealthDue(now) != (now.Sub(at) >= HealthTTL) {
			t.Fatalf("HealthDue %v after an attempt", now.Sub(at))
		}
		if !(Network{}).HealthDue(now) {
			t.Fatal("never attempted: due")
		}
	})
}

func propertyNetworkRecordHealthFresh(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		at := instantGen().Draw(t, "at")
		h := rapid.IntRange(0, 3).Draw(t, "h")
		n := Network{}.RecordHealth(h, at)
		if n.Health != h || !n.HealthFetchedAt.Equal(at) || n.HealthDue(at) {
			t.Fatalf("RecordHealth = %+v", n)
		}
	})
}

func propertyNetworkRecordHealthFailureKeepsLevel(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		at := instantGen().Draw(t, "at")
		n := Network{}.RecordHealth(rapid.IntRange(0, 3).Draw(t, "h"), t0)
		failed := n.RecordHealthFailure(at)
		if failed.Health != n.Health || !failed.HealthFetchedAt.Equal(n.HealthFetchedAt) || !failed.HealthAttemptAt.Equal(at) {
			t.Fatalf("RecordHealthFailure = %+v", failed)
		}
	})
}

func propertyNetworkRecordUsageResetsBackoff(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		n := Network{}
		for i := rapid.IntRange(0, 6).Draw(t, "failures"); i > 0; i-- {
			n = n.RecordUsageFailure(429, time.Duration(rapid.Int64Range(0, int64(time.Hour)).Draw(t, "ra")), t0)
		}
		at := instantGen().Draw(t, "at")
		ok := n.RecordUsage(quota.Set{}, at)
		if ok.UsageFailures != 0 || !ok.UsageRetryAfter.IsZero() || !ok.HasUsage || !ok.UsageFetchedAt.Equal(at) {
			t.Fatalf("RecordUsage = %+v", ok)
		}
	})
}

func propertyUpdateDueHourly(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		at, now := instantGen().Draw(t, "at"), instantGen().Draw(t, "now")
		if (Update{CheckedAt: at}).Due(now) != (now.Sub(at) >= UpdateInterval) {
			t.Fatal("Due")
		}
	})
}
