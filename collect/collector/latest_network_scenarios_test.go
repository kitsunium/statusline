// Written once from the design; this file is yours. The shells in
// gen_design.go delegate to what it holds; a body that still answers
// errNotImplemented is a stub.

package collector

import (
	"testing"

	"github.com/kitsunium/statusline/collect/state"
	"github.com/kitsunium/statusline/snapshot"
)

func givenLatestNetworkServesThePersistedFigures(t *testing.T) (LatestNetwork, LatestNetworkInput, func(*testing.T, LatestNetworkOutput)) {
	t.Helper()
	return NewLatestNetwork(&clock{now: t0}, &network{n: stateWithScoped()}), LatestNetworkInput{}, func(t *testing.T, out LatestNetworkOutput) {
		if !out.HasUsage || len(out.Usage.Scoped) != 1 || out.Health != snapshot.HealthDegraded {
			t.Errorf("out = %+v", out)
		}
	}
}

func givenLatestNetworkReadsAStaleSummaryAsUnknown(t *testing.T) (LatestNetwork, LatestNetworkInput, func(*testing.T, LatestNetworkOutput)) {
	t.Helper()
	return NewLatestNetwork(&clock{now: t0.Add(state.HealthMaxAge)}, &network{n: stateWithScoped()}), LatestNetworkInput{}, func(t *testing.T, out LatestNetworkOutput) {
		if out.Health != snapshot.HealthUnknown || !out.HasUsage {
			t.Errorf("out = %+v: past HealthMaxAge the health is unknown, the usage stays", out)
		}
	}
}
