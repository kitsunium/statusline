package refresh_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/kitsunium/statusline/collect/refresh"
	"github.com/kitsunium/statusline/collect/state"
	"github.com/kitsunium/statusline/quota"
	"github.com/kitsunium/statusline/snapshot"
)

var t0 = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

// world records every port call, in order, and answers from its fields.
type world struct {
	now      time.Time
	calls    []string
	saved    *state.Network
	stored   state.Network
	token    string
	usage    quota.Set
	usageErr error
	health   snapshot.Health
	pageErr  error
}

func (w *world) Now() time.Time { return w.now }

func (w *world) LoadNetwork() (state.Network, error) {
	w.calls = append(w.calls, "load")
	return w.stored, nil
}

func (w *world) SaveNetwork(n state.Network) error {
	w.calls = append(w.calls, "save")
	w.saved, w.stored = &n, n
	return nil
}

func (w *world) Token(context.Context) (string, error) {
	w.calls = append(w.calls, "token")
	if w.token == "" {
		return "", errors.New("no token")
	}
	return w.token, nil
}

func (w *world) Fetch(_ context.Context, token string) (quota.Set, error) {
	w.calls = append(w.calls, "usage:"+token)
	return w.usage, w.usageErr
}

// page is the status page side of the world.
type page struct{ w *world }

func (p page) Fetch(context.Context) (snapshot.Health, error) {
	p.w.calls = append(p.w.calls, "health")
	return p.w.health, p.w.pageErr
}

func newRefresher(w *world) *refresh.Refresher {
	return refresh.New(refresh.Deps{Clock: w, Store: w, Tokens: w, Usage: w, Status: page{w}})
}

func weekly(p int) quota.Set {
	return quota.Set{Weekly: quota.NewLimit(quota.KindWeekly, "weekly", p, t0.Add(time.Hour), quota.WeeklyWindow, quota.SourceAPI)}
}

// TestSequenceRefreshNetwork pins the order of design/sequences/refresh-network.yaml.
func TestSequenceRefreshNetwork(t *testing.T) {
	w := &world{now: t0, token: "tok", usage: weekly(40), health: snapshot.HealthOK}
	out, err := newRefresher(w).Execute(context.Background(), refresh.RefreshInput{})
	if err != nil || !out.UsageFetched || !out.HealthFetched {
		t.Fatalf("Execute() = %+v, %v", out, err)
	}
	if got, want := strings.Join(w.calls, ","), "load,token,usage:tok,health,save"; got != want {
		t.Errorf("calls = %s, want %s", got, want)
	}
}

func TestNotDueFetchesNothing(t *testing.T) {
	w := &world{now: t0, token: "tok", usage: weekly(40), health: snapshot.HealthOK}
	r := newRefresher(w)
	_, _ = r.Execute(context.Background(), refresh.RefreshInput{})
	w.calls = nil
	w.now = t0.Add(state.UsageTTL - time.Second)
	out, _ := r.Execute(context.Background(), refresh.RefreshInput{})
	if out.UsageFetched || out.HealthFetched || len(w.calls) != 0 {
		t.Errorf("before any TTL: %+v, calls %v", out, w.calls)
	}
	w.now = t0.Add(state.UsageTTL)
	out, _ = r.Execute(context.Background(), refresh.RefreshInput{})
	if !out.UsageFetched || out.HealthFetched {
		t.Errorf("at UsageTTL only usage is due: %+v", out)
	}
}

func TestRateLimitedPersistsAcrossARestart(t *testing.T) {
	w := &world{now: t0, token: "tok", usageErr: &state.RateLimited{RetryAfter: 10 * time.Minute}}
	_, _ = newRefresher(w).Execute(context.Background(), refresh.RefreshInput{})
	if w.saved == nil || !w.saved.UsageRetryAfter.Equal(t0.Add(10*time.Minute)) || w.saved.UsageStatus != 429 {
		t.Fatalf("saved = %+v, want RetryAfter persisted", w.saved)
	}
	// A new daemon reads the persisted state and keeps honouring it
	w.calls, w.usageErr, w.usage = nil, nil, weekly(10)
	w.now = t0.Add(5 * time.Minute)
	restarted := newRefresher(w)
	out, _ := restarted.Execute(context.Background(), refresh.RefreshInput{Force: true})
	for _, c := range w.calls {
		if strings.HasPrefix(c, "usage") || c == "token" {
			t.Fatalf("usage asked before Retry-After: %v", w.calls)
		}
	}
	if out.UsageFetched {
		t.Error("usage fetched before Retry-After")
	}
	w.now = t0.Add(10 * time.Minute)
	if out, _ := restarted.Execute(context.Background(), refresh.RefreshInput{}); !out.UsageFetched {
		t.Error("usage not fetched once Retry-After passed")
	}
}

func TestNoTokenKeepsTheLastPayload(t *testing.T) {
	w := &world{now: t0, token: "tok", usage: weekly(40)}
	r := newRefresher(w)
	_, _ = r.Execute(context.Background(), refresh.RefreshInput{})
	w.token, w.calls = "", nil
	w.now = t0.Add(state.UsageTTL)
	if out, _ := r.Execute(context.Background(), refresh.RefreshInput{}); out.UsageFetched {
		t.Error("fetched without a token")
	}
	for _, c := range w.calls {
		if strings.HasPrefix(c, "usage") {
			t.Fatalf("the API was called without a token: %v", w.calls)
		}
	}
	latest, _ := r.Latest().Execute(context.Background(), refresh.LatestInput{})
	if !latest.HasUsage || latest.Usage.Weekly.Percent != 40 {
		t.Errorf("latest = %+v, want the last payload kept", latest)
	}
	// Without a token the next look is one TTL later, not a back-off
	w.calls = nil
	w.now = w.now.Add(state.UsageTTL)
	_, _ = r.Execute(context.Background(), refresh.RefreshInput{})
	if len(w.calls) == 0 || w.calls[0] != "token" {
		t.Errorf("the token was not looked for again one TTL later: %v", w.calls)
	}
}

func TestHealthStaleReadsUnknown(t *testing.T) {
	w := &world{now: t0, health: snapshot.HealthDegraded}
	r := newRefresher(w)
	_, _ = r.Execute(context.Background(), refresh.RefreshInput{})
	w.pageErr = errors.New("unreachable")
	w.now = t0.Add(state.HealthMaxAge)
	_, _ = r.Execute(context.Background(), refresh.RefreshInput{})
	latest, _ := r.Latest().Execute(context.Background(), refresh.LatestInput{})
	if latest.Health != snapshot.HealthUnknown {
		t.Errorf("health %v past HealthMaxAge, want unknown", latest.Health)
	}
}

func TestLatestLoadsThePersistedFigures(t *testing.T) {
	w := &world{now: t0, stored: state.Network{}.RecordUsage(weekly(77), t0.Add(-time.Minute))}
	latest, _ := newRefresher(w).Latest().Execute(context.Background(), refresh.LatestInput{})
	if !latest.HasUsage || latest.Usage.Weekly.Percent != 77 {
		t.Errorf("latest = %+v, want the persisted payload before any tick", latest)
	}
}
