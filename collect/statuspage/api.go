// Package statuspage reads the provider's public status page summary.
//
// Exported API of design/domains/collect.yaml (collect/component/statuspage).
// This file stands in for the shells kit generates (api_gen.go) until
// `kit gen` is available: every exported symbol only delegates to its
// unexported twin.
package statuspage

import (
	"context"

	"github.com/kitsunium/statusline/snapshot"
)

// Client implements collect/port/status-page@v1.
type Client struct {
	url  string
	doer doer
}

// New returns a client for the summary; STATUSLINE_HEALTH_URL overrides it.
func New(getenv func(string) string) *Client { return newClient(getenv) }

// Fetch returns the aggregate level; an error leaves the caller's last level
// in place.
func (c *Client) Fetch(ctx context.Context) (snapshot.Health, error) { return c.fetch(ctx) }
