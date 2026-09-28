package usageapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kitsunium/statusline/collect/state"
	"github.com/kitsunium/statusline/quota"
)

const (
	// defaultURL is the OAuth usage endpoint.
	defaultURL string = "https://api.anthropic.com/api/oauth/usage"
	// urlEnv points the client at another endpoint.
	urlEnv string = "STATUSLINE_USAGE_URL"
	// betaHeader opts into the usage beta the endpoint requires.
	betaHeader string = "oauth-2025-04-20"
	// httpTimeout bounds one request; the daemon never blocks a render on it.
	httpTimeout time.Duration = 5 * time.Second
	// maxBody bounds the payload read; it is a few kilobytes.
	maxBody int64 = 1 << 20
)

// doer sends a request; *http.Client in production.
type doer interface {
	Do(req *http.Request) (*http.Response, error)
}

func newClient(getenv func(string) string) *Client {
	url := defaultURL
	if override := strings.TrimSpace(getenv(urlEnv)); override != "" {
		url = override
	}
	return &Client{url: url, doer: &http.Client{Timeout: httpTimeout}}
}

func (c *Client) fetch(ctx context.Context, token string) (quota.Set, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return quota.Set{}, fmt.Errorf("%w: %v", state.ErrUsageUnavailable, err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("anthropic-beta", betaHeader)
	resp, err := c.doer.Do(req)
	if err != nil {
		return quota.Set{}, fmt.Errorf("%w: %v", state.ErrUsageUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusTooManyRequests {
		return quota.Set{}, &state.RateLimited{RetryAfter: retryAfter(resp.Header.Get("Retry-After"), time.Now())}
	}
	if resp.StatusCode != http.StatusOK {
		return quota.Set{}, fmt.Errorf("%w: status %d", state.ErrUsageUnavailable, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return quota.Set{}, fmt.Errorf("%w: %v", state.ErrUsageUnavailable, err)
	}
	return decodeSet(body)
}

// decodeSet refuses a payload that does not decode: it must never
// masquerade as zero usage.
func decodeSet(body []byte) (quota.Set, error) {
	var parsed usageResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return quota.Set{}, fmt.Errorf("%w: undecodable payload", state.ErrUsageUnavailable)
	}
	return parsed.toLimitSet(), nil
}

// retryAfter reads delta-seconds or an HTTP date; zero when absent or bad.
func retryAfter(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if secs, err := strconv.Atoi(value); err == nil {
		return max(time.Duration(secs)*time.Second, 0)
	}
	if at, err := http.ParseTime(value); err == nil {
		return max(at.Sub(now), 0)
	}
	return 0
}
