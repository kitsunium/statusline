package state

import (
	"time"
)

// Back-off bounds of the usage API after a failure.
const (
	// maxBackoff caps the exponential back-off.
	maxBackoff time.Duration = time.Hour
	// maxRetryAfter caps a server's Retry-After: a garbage header must not
	// silence the quotas for days.
	maxRetryAfter time.Duration = 24 * time.Hour
)

// backoff doubles from UsageTTL up to maxBackoff.
func backoff(failures int) time.Duration {
	wait := UsageTTL
	for i := 1; i < failures && wait < maxBackoff; i++ {
		wait *= 2
	}
	return min(wait, maxBackoff)
}

// Error names the refusal with its code.
func (e *RateLimited) Error() string {
	return "statusline_rate_limited: usage API answered 429, retry after " + e.RetryAfter.String()
}
