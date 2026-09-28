package usageapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kitsunium/statusline/collect/state"
	"github.com/kitsunium/statusline/quota"
)

// serve answers every request with a status, headers and a body, and
// records the last request.
func serve(t *testing.T, status int, header map[string]string, body string, seen **http.Request) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if seen != nil {
			*seen = r
		}
		for k, v := range header {
			w.Header().Set(k, v)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return New(func(key string) string {
		if key == urlEnv {
			return srv.URL
		}
		return ""
	})
}

func TestFetchSendsTheTokenAndTheBeta(t *testing.T) {
	var seen *http.Request
	c := serve(t, http.StatusOK, nil, weeklyPlan, &seen)
	set, err := c.Fetch(context.Background(), "sk-ant-oat01-SYNTHETIC")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if got := seen.Header.Get("Authorization"); got != "Bearer sk-ant-oat01-SYNTHETIC" {
		t.Errorf("Authorization = %q", got)
	}
	if got := seen.Header.Get("anthropic-beta"); got != betaHeader {
		t.Errorf("anthropic-beta = %q", got)
	}
	if set.Weekly.Percent != 61 || !set.Weekly.IsValid() {
		t.Errorf("weekly = %+v, want 61 %%", set.Weekly)
	}
}

func TestFetchRateLimited(t *testing.T) {
	tests := []struct {
		name   string
		header map[string]string
		want   time.Duration
	}{
		{name: "delta seconds", header: map[string]string{"Retry-After": "120"}, want: 2 * time.Minute},
		{name: "no header", header: nil, want: 0},
		{name: "garbage", header: map[string]string{"Retry-After": "soon"}, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := serve(t, http.StatusTooManyRequests, tt.header, `{"error":"rate_limited"}`, nil)
			_, err := c.Fetch(context.Background(), "tok")
			var limited *state.RateLimited
			if !errors.As(err, &limited) {
				t.Fatalf("Fetch() error = %v, want *state.RateLimited", err)
			}
			if limited.RetryAfter != tt.want {
				t.Errorf("RetryAfter = %v, want %v", limited.RetryAfter, tt.want)
			}
		})
	}
}

func TestRetryAfterHTTPDate(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if got := retryAfter(now.Add(90*time.Second).Format(http.TimeFormat), now); got != 90*time.Second {
		t.Errorf("retryAfter(date) = %v, want 90s", got)
	}
	if got := retryAfter(now.Add(-time.Hour).Format(http.TimeFormat), now); got != 0 {
		t.Errorf("retryAfter(past date) = %v, want 0", got)
	}
}

func TestFetchRefusals(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "server error", status: http.StatusInternalServerError, body: "oops"},
		{name: "unauthorized", status: http.StatusUnauthorized, body: `{}`},
		{name: "undecodable", status: http.StatusOK, body: `["not","an","object"]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set, err := serve(t, tt.status, nil, tt.body, nil).Fetch(context.Background(), "tok")
			if !errors.Is(err, state.ErrUsageUnavailable) {
				t.Errorf("Fetch() error = %v, want ErrUsageUnavailable", err)
			}
			if set.HasTimed() {
				t.Errorf("a refused payload produced quotas: %+v", set)
			}
		})
	}
}

// TestDecodeObserved replays the observed shapes (design/evidence.yaml,
// usage-endpoint-shape).
func TestDecodeObserved(t *testing.T) {
	dir := filepath.Join("..", "..", "design", "evidence", "observed")
	read := func(name string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	set, err := decodeSet(read("usage-limits.json"))
	if err != nil {
		t.Fatalf("usage-limits.json: %v", err)
	}
	if set.Session.Percent != 72 || set.Weekly.Percent != 35 || len(set.Scoped) != 2 {
		t.Errorf("usage-limits.json = %+v", set)
	}
	if set.Scoped[1].Active {
		t.Error("is_active false was not kept")
	}
	if !set.Extra.IsValid() || set.Extra.UsedMinor != 9150 || set.Extra.Currency != "EUR" {
		t.Errorf("extra = %+v", set.Extra)
	}

	legacy, err := decodeSet(read("usage-legacy-buckets.json"))
	if err != nil || legacy.Session.Percent != 55 || legacy.Weekly.Percent != 20 || legacy.Extra.IsValid() {
		t.Errorf("usage-legacy-buckets.json = %+v, %v", legacy, err)
	}

	nullWeekly, err := decodeSet(read("usage-null-buckets.json"))
	if err != nil || nullWeekly.Weekly.IsValid() || nullWeekly.Session.Percent != 10 {
		t.Errorf("usage-null-buckets.json = %+v, %v: the weekly quota must stay absent", nullWeekly, err)
	}
	if nullWeekly.Weekly != (quota.Limit{}) {
		t.Errorf("absent weekly = %+v, want the zero Limit", nullWeekly.Weekly)
	}
}
