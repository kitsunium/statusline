// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package state

import (
	"time"

	"github.com/kitsunium/statusline/quota"
	"github.com/kitsunium/statusline/snapshot"
)

// currentHealth: past HealthMaxAge a green light would be a claim nobody
// has checked.
func (n Network) currentHealth(now time.Time) int {
	if n.HealthFetchedAt.IsZero() || now.Sub(n.HealthFetchedAt) >= HealthMaxAge {
		return snapshot.HealthUnknown
	}
	return n.Health
}

func (n Network) healthDue(now time.Time) bool {
	return n.HealthAttemptAt.IsZero() || now.Sub(n.HealthAttemptAt) >= HealthTTL
}

func (n Network) recordHealth(h int, now time.Time) Network {
	n.Health, n.HealthFetchedAt, n.HealthAttemptAt = h, now, now
	return n
}

func (n Network) recordHealthFailure(now time.Time) Network {
	n.HealthAttemptAt = now
	return n
}

func (n Network) recordUsage(set quota.Set, now time.Time) Network {
	n.Usage, n.HasUsage = set, true
	n.UsageFetchedAt, n.UsageAttemptAt = now, now
	n.UsageRetryAfter, n.UsageFailures, n.UsageStatus = time.Time{}, 0, 0
	return n
}

// recordUsageFailure keeps the last payload: a failure is not evidence of
// zero usage. The next attempt waits for the server's Retry-After when it
// gave one, else for the exponential back-off.
func (n Network) recordUsageFailure(status int, retryAfter time.Duration, now time.Time) Network {
	n.UsageAttemptAt, n.UsageStatus = now, status
	n.UsageFailures++
	wait := backoff(n.UsageFailures)
	if retryAfter > 0 {
		wait = min(max(retryAfter, UsageTTL), maxRetryAfter)
	}
	n.UsageRetryAfter = now.Add(wait)
	return n
}

// usageDue: never before a Retry-After, and at most once per UsageTTL
// counted from the last attempt, so that a failing endpoint is not asked on
// every tick.
func (n Network) usageDue(now time.Time) bool {
	if now.Before(n.UsageRetryAfter) {
		return false
	}
	return n.UsageAttemptAt.IsZero() || now.Sub(n.UsageAttemptAt) >= UsageTTL
}
