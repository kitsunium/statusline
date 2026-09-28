package statuspage

import (
	"context"
	"net/http"
	"os"
)

// statusPage is where the summary is fetched from, and how.
type statusPage struct {
	url  string
	doer doer
}

// doer sends a request; *http.Client in production.
type doer interface {
	Do(req *http.Request) (*http.Response, error)
}

func newStatusPage() *StatusPage { return &StatusPage{newPage(os.Getenv)} }

// fetch returns the aggregate level; an error leaves the caller's last one.
func (a *StatusPage) fetch(ctx context.Context) (int, error) { return a.get(ctx) }
