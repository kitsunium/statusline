// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package collector

import (
	"context"
	"errors"
	"net/http"

	"github.com/kitsunium/statusline/collect/state"
)

// execute fetches what is due and persists the bookkeeping after any
// attempt, so that a restarted daemon keeps honouring a Retry-After.
func (h *RefreshNetworkHandler) execute(ctx context.Context, in RefreshNetworkInput) (RefreshNetworkOutput, error) {
	now, err := h.clock.Now(ctx)
	if err != nil {
		return RefreshNetworkOutput{}, err
	}
	n, loadErr := h.networkStore.LoadNetwork(ctx)
	var out RefreshNetworkOutput
	attempted := false
	if in.Force && !now.Before(n.UsageRetryAfter) || n.UsageDue(now) {
		n, out.UsageFetched = h.refreshUsage(ctx, n)
		attempted = true
	}
	if in.Force || n.HealthDue(now) {
		n, out.HealthFetched = h.refreshHealth(ctx, n)
		attempted = true
	}
	if !attempted {
		return out, loadErr
	}
	return out, errors.Join(loadErr, h.networkStore.SaveNetwork(ctx, n))
}

// refreshUsage reads the token for this call only. Without one the account
// has no API enrichment: nothing is asked, and the attempt is recorded
// without a back-off so that a fresh login shows within a minute.
func (h *RefreshNetworkHandler) refreshUsage(ctx context.Context, n state.Network) (state.Network, bool) {
	now, _ := h.clock.Now(ctx)
	token, err := h.tokenStore.Token(ctx)
	if err != nil || token == "" {
		n.UsageAttemptAt = now
		return n, false
	}
	set, err := h.usageAPI.Fetch(ctx, token)
	var limited *state.RateLimited
	switch {
	case errors.As(err, &limited):
		return n.RecordUsageFailure(http.StatusTooManyRequests, limited.RetryAfter, now), false
	case err != nil:
		return n.RecordUsageFailure(0, 0, now), false
	}
	return n.RecordUsage(set, now), true
}

// refreshHealth keeps the last level on a failure; it ages out on its own.
func (h *RefreshNetworkHandler) refreshHealth(ctx context.Context, n state.Network) (state.Network, bool) {
	now, _ := h.clock.Now(ctx)
	level, err := h.statusPage.Fetch(ctx)
	if err != nil {
		return n.RecordHealthFailure(now), false
	}
	return n.RecordHealth(level, now), true
}
