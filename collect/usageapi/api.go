// Package usageapi fetches the account's quotas from the OAuth usage
// endpoint.
//
// Exported API of design/domains/collect.yaml (collect/component/usageapi).
// This file stands in for the shells kit generates (api_gen.go) until
// `kit gen` is available: every exported symbol only delegates to its
// unexported twin.
package usageapi

import (
	"context"

	"github.com/kitsunium/statusline/quota"
)

// Client implements collect/port/usage-api@v1.
type Client struct {
	url  string
	doer doer
}

// New returns a client for the endpoint; STATUSLINE_USAGE_URL overrides it
// (tests, proxies).
func New(getenv func(string) string) *Client { return newClient(getenv) }

// Fetch asks the endpoint once. A 429 is a *state.RateLimited; any other
// failure wraps state.ErrUsageUnavailable. The token is sent to this
// endpoint only, and never kept.
func (c *Client) Fetch(ctx context.Context, token string) (quota.Set, error) {
	return c.fetch(ctx, token)
}
