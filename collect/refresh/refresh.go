package refresh

import (
	"context"
	"errors"
	"net/http"

	"github.com/kitsunium/statusline/collect/state"
)

// execute fetches what is due and persists the bookkeeping after any
// attempt, so that a restarted daemon keeps honouring a Retry-After.
func (r *Refresher) execute(ctx context.Context, in RefreshInput) (RefreshOutput, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := r.deps.Clock.Now()
	loadErr := r.load()
	var out RefreshOutput
	attempted := false
	if in.Force && !now.Before(r.net.UsageRetryAfter) || r.net.UsageDue(now) {
		out.UsageFetched, attempted = r.refreshUsage(ctx), true
	}
	if in.Force || r.net.HealthDue(now) {
		out.HealthFetched, attempted = r.refreshHealth(ctx), true
	}
	if !attempted {
		return out, loadErr
	}
	return out, errors.Join(loadErr, r.deps.Store.SaveNetwork(r.net))
}

// refreshUsage reads the token for this call only. Without one the account
// has no API enrichment: nothing is asked, and the attempt is recorded
// without a back-off so that a fresh login shows within a minute.
func (r *Refresher) refreshUsage(ctx context.Context) bool {
	now := r.deps.Clock.Now()
	token, err := r.deps.Tokens.Token(ctx)
	if err != nil || token == "" {
		r.net.UsageAttemptAt = now
		return false
	}
	set, err := r.deps.Usage.Fetch(ctx, token)
	var limited *state.RateLimited
	switch {
	case errors.As(err, &limited):
		r.net = r.net.RecordUsageFailure(http.StatusTooManyRequests, limited.RetryAfter, now)
		return false
	case err != nil:
		r.net = r.net.RecordUsageFailure(0, 0, now)
		return false
	}
	r.net = r.net.RecordUsage(set, now)
	return true
}

// refreshHealth keeps the last level on a failure; it ages out on its own.
func (r *Refresher) refreshHealth(ctx context.Context) bool {
	now := r.deps.Clock.Now()
	h, err := r.deps.Status.Fetch(ctx)
	if err != nil {
		r.net = r.net.RecordHealthFailure(now)
		return false
	}
	r.net = r.net.RecordHealth(h, now)
	return true
}

func (r *Refresher) latest(_ context.Context, _ LatestInput) (LatestOutput, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	_ = r.load()
	now := r.deps.Clock.Now()
	return LatestOutput{
		Usage:    r.net.Usage,
		HasUsage: r.net.HasUsage,
		Health:   r.net.CurrentHealth(now),
		Network:  r.net,
	}, nil
}

// load reads the persisted bookkeeping once: a restarted daemon serves the
// last figures at once and keeps honouring a Retry-After. The caller holds mu.
func (r *Refresher) load() error {
	if r.loaded {
		return nil
	}
	r.loaded = true
	n, err := r.deps.Store.LoadNetwork()
	if err != nil {
		return err
	}
	r.net = n
	return nil
}
