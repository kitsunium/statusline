package usageapi

import (
	"context"
	"net/http"
	"os"

	"github.com/kitsunium/statusline/quota"
)

// usage is where the payload is fetched from, and how.
type usage struct {
	url  string
	doer doer
}

// doer sends a request; *http.Client in production.
type doer interface {
	Do(req *http.Request) (*http.Response, error)
}

func newUsage() *Usage { return &Usage{newEndpoint(os.Getenv)} }

// fetch asks the endpoint once; the token goes to it only, and is not kept.
func (a *Usage) fetch(ctx context.Context, token string) (quota.Set, error) { return a.get(ctx, token) }
