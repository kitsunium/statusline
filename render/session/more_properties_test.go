package session

import (
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// seeded logs the design seed: rapid v1.2 takes its own from -rapid.seed.
func seeded(t *testing.T, seed uint64) {
	t.Helper()
	t.Logf("design seed %d (rapid v1.2 takes its own seed from -rapid.seed)", seed)
}

func pct(t *rapid.T, label string) *float64 {
	if !rapid.Bool().Draw(t, "has "+label) {
		return nil
	}
	v := rapid.Float64Range(-50, 250).Draw(t, label)
	return &v
}

func payloadGen() *rapid.Generator[Payload] {
	return rapid.Custom(func(t *rapid.T) Payload {
		var p Payload
		p.Model.DisplayName = rapid.StringMatching(`([A-Z][a-z]{0,6}( [0-9.]{1,4}( \(1M context\))?)?)?`).Draw(t, "model")
		p.Workspace.CurrentDir = rapid.StringMatching(`(/[a-z]{1,5}){0,3}`).Draw(t, "dir")
		p.ContextWindow.TotalInputTokens = rapid.IntRange(0, 2_000_000).Draw(t, "in")
		p.ContextWindow.TotalOutputTokens = rapid.IntRange(0, 200_000).Draw(t, "out")
		p.ContextWindow.ContextWindowSize = rapid.IntRange(0, 1_000_000).Draw(t, "size")
		p.ContextWindow.UsedPercentage = pct(t, "used")
		if rapid.Bool().Draw(t, "five") {
			p.RateLimits.FiveHour = &PayloadRateLimit{UsedPercentage: pct(t, "5h used"), Utilization: pct(t, "5h util")}
		}
		return p
	})
}

func propertyModelInfoFullNameStartsWithName(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		p := payloadGen().Draw(t, "p")
		m := p.ModelInfo()
		if !strings.HasPrefix(m.FullName(), m.Name) || (m.Version == "") != (m.FullName() == m.Name) {
			t.Fatalf("FullName(%+v) = %q", m, m.FullName())
		}
	})
}

func propertyModelInfoShortNamePrefixOfFull(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		p := payloadGen().Draw(t, "p")
		m := p.ModelInfo()
		if !strings.HasPrefix(m.FullName(), m.ShortName()) || strings.Contains(m.ShortName(), "(") {
			t.Fatalf("ShortName(%+v) = %q", m, m.ShortName())
		}
	})
}

func propertyPayloadContextWindowSizePositive(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		p := payloadGen().Draw(t, "p")
		if s := p.ContextWindowSize(); s <= 0 || (p.ContextWindow.ContextWindowSize > 0 && s != p.ContextWindow.ContextWindowSize) {
			t.Fatalf("ContextWindowSize = %d", s)
		}
	})
}

func propertyPayloadModelInfoDefaultName(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		p := payloadGen().Draw(t, "p")
		if p.Model.DisplayName == "" && p.ModelInfo().Name != defaultModelName {
			t.Fatal("no display name must give the default model")
		}
		if p.Model.DisplayName != "" && p.ModelInfo().FullName() != p.Model.DisplayName {
			t.Fatalf("FullName %q lost the display name %q", p.ModelInfo().FullName(), p.Model.DisplayName)
		}
	})
}

func propertyPayloadProgressBounded(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		p := payloadGen().Draw(t, "p")
		if pr := p.Progress(); pr.Percent > 100 {
			t.Fatalf("Progress = %d", pr.Percent)
		}
	})
}

func propertyPayloadStdinLimitsContextAlways(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		p := payloadGen().Draw(t, "p")
		set := p.StdinLimits()
		if !set.Context.IsValid() || set.Weekly.IsValid() {
			t.Fatalf("StdinLimits = %+v", set)
		}
		if _, ok := p.RateLimits.FiveHour.Percent(); !ok && set.Session.Percent != 0 {
			t.Fatal("an absent bucket produced a quota")
		}
	})
}

func propertyPayloadTotalTokensSum(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		p := payloadGen().Draw(t, "p")
		if p.TotalTokens() != p.ContextWindow.TotalInputTokens+p.ContextWindow.TotalOutputTokens {
			t.Fatal("TotalTokens")
		}
	})
}

func propertyPayloadWorkingDirNeverEmpty(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		p := payloadGen().Draw(t, "p")
		if d := p.WorkingDir(); d == "" || (p.Workspace.CurrentDir != "" && d != p.Workspace.CurrentDir) {
			t.Fatalf("WorkingDir = %q", d)
		}
	})
}

func propertyPayloadRateLimitPercentPrefersUsed(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		r := &PayloadRateLimit{UsedPercentage: pct(t, "used"), Utilization: pct(t, "util")}
		got, ok := r.Percent()
		switch {
		case r.UsedPercentage != nil:
			if !ok || got != int(*r.UsedPercentage) {
				t.Fatal("used_percentage must win")
			}
		case r.Utilization != nil:
			if !ok || got != int(*r.Utilization) {
				t.Fatal("utilization is the fallback")
			}
		default:
			if ok {
				t.Fatal("no field, no percentage")
			}
		}
		var nilLimit *PayloadRateLimit
		if _, ok := nilLimit.Percent(); ok {
			t.Fatal("a nil bucket has a percentage")
		}
	})
}

func propertyEffortStepsMatchesScale(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		i := rapid.IntRange(0, EffortSteps()-1).Draw(t, "i")
		if rank, ok := EffortRank(effortScale[i]); !ok || rank != i+1 {
			t.Fatalf("EffortRank(%q) = %d", effortScale[i], rank)
		}
	})
}
