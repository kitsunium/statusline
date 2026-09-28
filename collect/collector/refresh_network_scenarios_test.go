// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package collector

import (
	"net/http"
	"testing"
	"time"

	"github.com/kitsunium/statusline/collect/state"
	"github.com/kitsunium/statusline/snapshot"
)

func refreshing(clk *clock, net *network, e *endpoints) RefreshNetwork {
	return NewRefreshNetwork(clk, net, e, e, page{e})
}

func givenRefreshNetworkFetchesWhatIsDue(t *testing.T) (RefreshNetwork, RefreshNetworkInput, func(*testing.T, RefreshNetworkOutput)) {
	t.Helper()
	net, e := &network{}, &endpoints{token: "tok", usage: weekly(40), health: snapshot.HealthOK}
	return refreshing(&clock{now: t0}, net, e), RefreshNetworkInput{}, func(t *testing.T, out RefreshNetworkOutput) {
		if !out.UsageFetched || !out.HealthFetched || net.n.Usage.Weekly.Percent != 40 || net.n.Health != snapshot.HealthOK {
			t.Errorf("out = %+v, saved %+v", out, net.n)
		}
	}
}

func givenRefreshNetworkFetchesNothingBeforeItsTtl(t *testing.T) (RefreshNetwork, RefreshNetworkInput, func(*testing.T, RefreshNetworkOutput)) {
	t.Helper()
	clk, net, e := &clock{now: t0}, &network{}, &endpoints{token: "tok", usage: weekly(40), health: snapshot.HealthOK}
	uc := refreshing(clk, net, e)
	if _, err := uc.Execute(t.Context(), RefreshNetworkInput{}); err != nil {
		t.Fatal(err)
	}
	clk.now = t0.Add(state.UsageTTL - time.Second)
	before := e.total()
	return uc, RefreshNetworkInput{}, func(t *testing.T, out RefreshNetworkOutput) {
		if out.UsageFetched || out.HealthFetched || e.total() != before {
			t.Errorf("before any TTL: %+v, %d endpoint calls", out, e.total()-before)
		}
	}
}

func givenRefreshNetworkHonoursARetryAfterAcrossARestart(t *testing.T) (RefreshNetwork, RefreshNetworkInput, func(*testing.T, RefreshNetworkOutput)) {
	t.Helper()
	clk, net := &clock{now: t0}, &network{}
	limited := &endpoints{token: "tok", usageErr: &state.RateLimited{RetryAfter: 10 * time.Minute}}
	if _, err := refreshing(clk, net, limited).Execute(t.Context(), RefreshNetworkInput{}); err != nil {
		t.Fatal(err)
	}
	if !net.n.UsageRetryAfter.Equal(t0.Add(10*time.Minute)) || net.n.UsageStatus != http.StatusTooManyRequests {
		t.Fatalf("Retry-After not persisted: %+v", net.n)
	}
	// A new daemon on the same persisted state, five minutes later, forced
	clk.now = t0.Add(5 * time.Minute)
	fresh := &endpoints{token: "tok", usage: weekly(10)}
	return refreshing(clk, net, fresh), RefreshNetworkInput{Force: true}, func(t *testing.T, out RefreshNetworkOutput) {
		if out.UsageFetched || fresh.n["usage"] != 0 || fresh.n["token"] != 0 {
			t.Errorf("usage asked before Retry-After: %+v, %v", out, fresh.n)
		}
	}
}

func givenRefreshNetworkKeepsTheLastPayloadWithoutAToken(t *testing.T) (RefreshNetwork, RefreshNetworkInput, func(*testing.T, RefreshNetworkOutput)) {
	t.Helper()
	clk, net, e := &clock{now: t0}, &network{}, &endpoints{token: "tok", usage: weekly(40)}
	uc := refreshing(clk, net, e)
	if _, err := uc.Execute(t.Context(), RefreshNetworkInput{}); err != nil {
		t.Fatal(err)
	}
	e.token = ""
	clk.now = t0.Add(state.UsageTTL)
	usageCalls := e.n["usage"]
	return uc, RefreshNetworkInput{}, func(t *testing.T, out RefreshNetworkOutput) {
		if out.UsageFetched || e.n["usage"] != usageCalls {
			t.Errorf("the API was called without a token: %+v", out)
		}
		if !net.n.HasUsage || net.n.Usage.Weekly.Percent != 40 || net.n.UsageFailures != 0 {
			t.Errorf("the last payload must be kept, without a back-off: %+v", net.n)
		}
	}
}
