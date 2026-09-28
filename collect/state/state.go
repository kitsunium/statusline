package state

import (
	"errors"
	"time"

	"github.com/kitsunium/statusline/ipc"
	"github.com/kitsunium/statusline/quota"
	"github.com/kitsunium/statusline/snapshot"
)

// Back-off bounds of the usage API after a failure.
const (
	// maxBackoff caps the exponential back-off.
	maxBackoff time.Duration = time.Hour
	// maxRetryAfter caps a server's Retry-After: a garbage header must not
	// silence the quotas for days.
	maxRetryAfter time.Duration = 24 * time.Hour
)

// entry is one session key's state.
type entry struct {
	snap        snapshot.Snapshot
	collectedAt time.Time
	requestedAt time.Time
	has         bool
}

func newRegistry() *Registry {
	return &Registry{entries: make(map[ipc.Key]*entry)}
}

func (r *Registry) get(key ipc.Key, now time.Time) (snapshot.Snapshot, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.entries[key]
	if !ok || !e.has || now.Sub(e.collectedAt) >= Freshness || now.Before(e.collectedAt) {
		return snapshot.Snapshot{}, false
	}
	return e.snap, true
}

func (r *Registry) put(key ipc.Key, snap snapshot.Snapshot, now time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e := r.entry(key)
	e.snap, e.collectedAt, e.has = snap, now, true
}

func (r *Registry) touch(key ipc.Key, now time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entry(key).requestedAt = now
	if now.After(r.last) {
		r.last = now
	}
}

// entry returns the key's entry, created on first use; the caller holds mu.
func (r *Registry) entry(key ipc.Key) *entry {
	e, ok := r.entries[key]
	if !ok {
		e = &entry{}
		r.entries[key] = e
	}
	return e
}

// evict drops exactly the keys whose last request is EvictAfter old.
func (r *Registry) evict(now time.Time) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for key, e := range r.entries {
		if now.Sub(e.requestedAt) >= EvictAfter {
			delete(r.entries, key)
			n++
		}
	}
	return n
}

func (r *Registry) len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.entries)
}

func (r *Registry) lastRequest() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.last
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

func (n Network) healthDue(now time.Time) bool {
	return n.HealthAttemptAt.IsZero() || now.Sub(n.HealthAttemptAt) >= HealthTTL
}

// currentHealth: past HealthMaxAge a green light would be a claim nobody
// has checked.
func (n Network) currentHealth(now time.Time) snapshot.Health {
	if n.HealthFetchedAt.IsZero() || now.Sub(n.HealthFetchedAt) >= HealthMaxAge {
		return snapshot.HealthUnknown
	}
	return n.Health
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

// backoff doubles from UsageTTL up to maxBackoff.
func backoff(failures int) time.Duration {
	wait := UsageTTL
	for i := 1; i < failures && wait < maxBackoff; i++ {
		wait *= 2
	}
	return min(wait, maxBackoff)
}

func (n Network) recordHealth(h snapshot.Health, now time.Time) Network {
	n.Health, n.HealthFetchedAt, n.HealthAttemptAt = h, now, now
	return n
}

func (n Network) recordHealthFailure(now time.Time) Network {
	n.HealthAttemptAt = now
	return n
}

func (u Update) due(now time.Time) bool {
	return u.CheckedAt.IsZero() || now.Sub(u.CheckedAt) >= UpdateInterval
}

// errUsageUnavailable carries the product's error code.
var errUsageUnavailable = errors.New("statusline_usage_unavailable: usage API unavailable")

func (e *RateLimited) error() string {
	return "statusline_rate_limited: usage API answered 429, retry after " + e.RetryAfter.String()
}
