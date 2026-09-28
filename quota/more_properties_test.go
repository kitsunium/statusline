package quota

import (
	"testing"

	"pgregory.net/rapid"
)

// seeded logs the design seed: rapid v1.2 takes its own from -rapid.seed.
func seeded(t *testing.T, seed uint64) {
	t.Helper()
	t.Logf("design seed %d (rapid v1.2 takes its own seed from -rapid.seed)", seed)
}

func propertyExtraIsValidRequiresEnabled(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		e := Extra{Enabled: rapid.Bool().Draw(t, "on"), Source: rapid.SampledFrom([]Source{SourceAPI, SourceNone}).Draw(t, "src")}
		if e.IsValid() != (e.Enabled && e.Source != SourceNone) {
			t.Fatalf("IsValid(%+v) = %v", e, e.IsValid())
		}
	})
}

func propertyLimitCursorPositionBounded(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		l, now := limitGen(KindSession).Draw(t, "l"), nowGen().Draw(t, "now")
		if c := l.CursorPosition(now); c < 0 || c > 100 {
			t.Fatalf("CursorPosition = %d", c)
		}
	})
}

func propertyLimitHasWindowNeedsBoth(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		l := limitGen(KindWeekly).Draw(t, "l")
		if l.HasWindow() != (l.Window > 0 && !l.ResetsAt.IsZero()) {
			t.Fatalf("HasWindow(%+v) = %v", l, l.HasWindow())
		}
	})
}

func propertyLimitIsOnTrackMatchesPace(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		l, now := limitGen(KindSession).Draw(t, "l"), nowGen().Draw(t, "now")
		if l.IsOnTrack(now) != (l.Pace(now) <= 0) {
			t.Fatal("IsOnTrack disagrees with Pace")
		}
	})
}

func propertyLimitIsValidNeedsSource(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		kind := rapid.SampledFrom([]Kind{KindContext, KindSession, KindWeekly}).Draw(t, "kind")
		l := limitGen(kind).Draw(t, "l")
		if l.Source == SourceNone && l.IsValid() {
			t.Fatal("a limit without a source is valid")
		}
		if kind != KindContext && l.ResetsAt.IsZero() && l.IsValid() {
			t.Fatal("a timed limit without a reset is valid")
		}
	})
}

func propertyLimitPaceAgainstCursor(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		l, now := limitGen(KindWeekly).Draw(t, "l"), nowGen().Draw(t, "now")
		if l.Pace(now) != l.Percent-l.CursorPosition(now) {
			t.Fatal("Pace is not Percent minus the cursor")
		}
	})
}

func propertyLimitProgressKeepsPercent(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		l := limitGen(KindWeekly).Draw(t, "l")
		if l.Progress().Percent != l.Percent {
			t.Fatal("Progress changed the percentage")
		}
	})
}

func propertyLimitProjectedNonNegative(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		l, now := limitGen(KindSession).Draw(t, "l"), nowGen().Draw(t, "now")
		if p := l.Projected(now); p < 0 || (l.Elapsed(now) >= 1 && p != l.Percent) {
			t.Fatalf("Projected = %d for %d %% at elapsed %v", p, l.Percent, l.Elapsed(now))
		}
	})
}

func propertyLimitRemainingNeverNegative(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		l, now := limitGen(KindSession).Draw(t, "l"), nowGen().Draw(t, "now")
		r := l.Remaining(now)
		if r < 0 || (!l.ResetsAt.IsZero() && now.Before(l.ResetsAt) && r != l.ResetsAt.Sub(now)) {
			t.Fatalf("Remaining = %v", r)
		}
	})
}

func propertyProgressLevelMonotonic(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		a := rapid.IntRange(-10, 110).Draw(t, "a")
		b := rapid.IntRange(a, 110).Draw(t, "b")
		la, lb := Progress{Percent: a}.Level(), Progress{Percent: b}.Level()
		if la > lb || la < LevelLow || lb > LevelCritical {
			t.Fatalf("Level(%d) = %d, Level(%d) = %d", a, la, b, lb)
		}
	})
}

func propertySetHasTimedMatchesTimed(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		s := setGen().Draw(t, "s")
		if s.HasTimed() != (len(s.Timed()) > 0) {
			t.Fatal("HasTimed disagrees with Timed")
		}
	})
}

func propertySetScopedForSubsetOfScoped(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		s := setGen().Draw(t, "s")
		model := rapid.StringMatching(`[A-Za-z ]{0,10}`).Draw(t, "model")
		got := s.ScopedFor(model)
		if len(got) > len(s.Scoped) {
			t.Fatal("ScopedFor returned more than the scoped quotas")
		}
		for _, l := range got {
			if !l.IsValid() {
				t.Fatal("ScopedFor returned an invalid quota")
			}
		}
	})
}

func propertySetTimedOnlyValid(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		s := setGen().Draw(t, "s")
		for _, l := range s.Timed() {
			if !l.IsValid() || l.Kind == KindContext {
				t.Fatalf("Timed returned %+v", l)
			}
		}
	})
}
