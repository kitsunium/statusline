package statuspage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kitsunium/statusline/snapshot"
)

const (
	// defaultURL is the Statuspage summary of the provider's services.
	defaultURL string = "https://status.claude.com/api/v2/summary.json"
	// urlEnv points the client at another summary.
	urlEnv string = "STATUSLINE_HEALTH_URL"
	// httpTimeout bounds one fetch.
	httpTimeout time.Duration = 5 * time.Second
	// maxBody bounds the payload read; the summary is a few kilobytes.
	maxBody int64 = 1 << 20
	// excludedComponent serves a separate public-sector deployment, not
	// this account.
	excludedComponent string = "government"
)

// summary is the part of the payload read here.
type summary struct {
	Components []component `json:"components"`
}

// component is one service on the page.
type component struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Group  bool   `json:"group"`
}

func newPage(getenv func(string) string) statusPage {
	url := defaultURL
	if override := strings.TrimSpace(getenv(urlEnv)); override != "" {
		url = override
	}
	return statusPage{url: url, doer: &http.Client{Timeout: httpTimeout}}
}

func (c statusPage) get(ctx context.Context) (snapshot.Health, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return snapshot.HealthUnknown, fmt.Errorf("status page: %w", err)
	}
	resp, err := c.doer.Do(req)
	if err != nil {
		return snapshot.HealthUnknown, fmt.Errorf("status page: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return snapshot.HealthUnknown, fmt.Errorf("status page: HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return snapshot.HealthUnknown, fmt.Errorf("status page: %w", err)
	}
	return classify(data)
}

// classify counts the individual services this account depends on: a group
// only restates its members, and the public-sector deployment is not ours.
// An undecodable summary is refused, never stored as a level.
func classify(data []byte) (snapshot.Health, error) {
	var parsed summary
	if err := json.Unmarshal(data, &parsed); err != nil {
		return snapshot.HealthUnknown, fmt.Errorf("status page: decode: %w", err)
	}
	states := make([]string, 0, len(parsed.Components))
	for _, comp := range parsed.Components {
		if comp.Group || strings.Contains(strings.ToLower(comp.Name), excludedComponent) {
			continue
		}
		states = append(states, comp.Status)
	}
	return snapshot.ClassifyHealth(states), nil
}
