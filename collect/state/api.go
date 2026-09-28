// Package state is the daemon's state: sessions by key with their eviction,
// the network bookkeeping with its 429 back-off, the update bookkeeping with
// its bad version.
//
// Exported API of design/domains/collect.yaml (collect/component/state).
// This file stands in for the shells kit generates (api_gen.go) until
// `kit gen` is available: every exported symbol only delegates to its
// unexported twin.
package state

import (
	"sync"
	"time"

	"github.com/kitsunium/statusline/ipc"
	"github.com/kitsunium/statusline/quota"
	"github.com/kitsunium/statusline/snapshot"
)

// EvictAfter is how long a key may go unrequested before it is dropped.
const EvictAfter time.Duration = 60 * time.Second

// Freshness is how old a snapshot may be and still be served as is.
const Freshness time.Duration = 750 * time.Millisecond

// UsageTTL is how often the usage API is asked at most.
const UsageTTL time.Duration = 60 * time.Second

// HealthTTL is how often the status page is asked at most.
const HealthTTL time.Duration = 2 * time.Minute

// HealthMaxAge is how old a summary may be and still be drawn.
const HealthMaxAge time.Duration = 15 * time.Minute

// UpdateInterval is how often a new release is looked for.
const UpdateInterval time.Duration = time.Hour

// HostSession is what the host's session registry says about a session.
type HostSession struct {
	PID     int
	Working bool
}

// Release is a release found on the release host.
type Release struct {
	Version string
	Asset   string
}

// Registry holds session states by key; safe for concurrent use.
type Registry struct {
	mu      sync.Mutex
	entries map[ipc.Key]*entry
	last    time.Time
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry { return newRegistry() }

// Get returns the key's snapshot when it is younger than Freshness.
func (r *Registry) Get(key ipc.Key, now time.Time) (snapshot.Snapshot, bool) { return r.get(key, now) }

// Put stores a freshly collected snapshot.
func (r *Registry) Put(key ipc.Key, snap snapshot.Snapshot, now time.Time) { r.put(key, snap, now) }

// Touch records a request for the key.
func (r *Registry) Touch(key ipc.Key, now time.Time) { r.touch(key, now) }

// Evict drops the keys unrequested for EvictAfter and returns how many.
func (r *Registry) Evict(now time.Time) int { return r.evict(now) }

// Len returns how many keys are held.
func (r *Registry) Len() int { return r.len() }

// LastRequest returns when any key was last requested.
func (r *Registry) LastRequest() time.Time { return r.lastRequest() }

// Network is the network bookkeeping, persisted so that a restarted daemon
// keeps honouring a 429.
type Network struct {
	Usage           quota.Set       `json:"usage"`
	HasUsage        bool            `json:"has_usage,omitempty"`
	UsageFetchedAt  time.Time       `json:"usage_fetched_at,omitzero"`
	UsageAttemptAt  time.Time       `json:"usage_attempt_at,omitzero"`
	UsageRetryAfter time.Time       `json:"usage_retry_after,omitzero"`
	UsageFailures   int             `json:"usage_failures,omitempty"`
	UsageStatus     int             `json:"usage_status,omitempty"`
	Health          snapshot.Health `json:"health,omitempty"`
	HealthFetchedAt time.Time       `json:"health_fetched_at,omitzero"`
	HealthAttemptAt time.Time       `json:"health_attempt_at,omitzero"`
}

// UsageDue reports whether the usage API may be asked now.
func (n Network) UsageDue(now time.Time) bool { return n.usageDue(now) }

// HealthDue reports whether the status page may be asked now.
func (n Network) HealthDue(now time.Time) bool { return n.healthDue(now) }

// CurrentHealth returns the level, unknown past HealthMaxAge.
func (n Network) CurrentHealth(now time.Time) snapshot.Health { return n.currentHealth(now) }

// RecordUsage records a fetched payload.
func (n Network) RecordUsage(set quota.Set, now time.Time) Network { return n.recordUsage(set, now) }

// RecordUsageFailure records a refused or failed request; a positive
// retryAfter is the server's own Retry-After.
func (n Network) RecordUsageFailure(status int, retryAfter time.Duration, now time.Time) Network {
	return n.recordUsageFailure(status, retryAfter, now)
}

// RecordHealth records a fetched summary.
func (n Network) RecordHealth(h snapshot.Health, now time.Time) Network {
	return n.recordHealth(h, now)
}

// RecordHealthFailure records a failed summary fetch.
func (n Network) RecordHealthFailure(now time.Time) Network { return n.recordHealthFailure(now) }

// Update is the update bookkeeping.
type Update struct {
	CheckedAt  time.Time `json:"checked_at,omitzero"`
	BadVersion string    `json:"bad_version,omitempty"`
	Failures   int       `json:"failures,omitempty"`
	Installed  string    `json:"installed,omitempty"`
}

// Due reports whether a release should be looked for now.
func (u Update) Due(now time.Time) bool { return u.due(now) }

// RateLimited is the usage API's HTTP 429. RetryAfter is the server's
// Retry-After, zero when it gave none.
type RateLimited struct {
	RetryAfter time.Duration
}

// Error names the refusal with its code.
func (e *RateLimited) Error() string { return e.error() }

// ErrUsageUnavailable is any other failure to obtain the usage payload.
var ErrUsageUnavailable = errUsageUnavailable
