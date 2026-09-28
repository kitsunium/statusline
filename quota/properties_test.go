// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package quota

import (
	"encoding/json"
	"testing"
	"time"

	"pgregory.net/rapid"
)

var epoch = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// limitGen draws any limit, valid or not, around epoch.
func limitGen(kind Kind) *rapid.Generator[Limit] {
	return rapid.Custom(func(t *rapid.T) Limit {
		percent := rapid.IntRange(-500, 500).Draw(t, "percent")
		window := time.Duration(rapid.Int64Range(0, int64(8*24*time.Hour)).Draw(t, "window"))
		var reset time.Time
		if rapid.Bool().Draw(t, "hasReset") {
			reset = epoch.Add(time.Duration(rapid.Int64Range(-int64(10*24*time.Hour), int64(10*24*time.Hour)).Draw(t, "reset")))
		}
		source := rapid.SampledFrom([]Source{SourceStdin, SourceAPI, SourceNone}).Draw(t, "source")
		return NewLimit(kind, "l", percent, reset, window, source)
	})
}

func nowGen() *rapid.Generator[time.Time] {
	return rapid.Custom(func(t *rapid.T) time.Time {
		return epoch.Add(time.Duration(rapid.Int64Range(-int64(10*24*time.Hour), int64(10*24*time.Hour)).Draw(t, "now")))
	})
}

// propertyNewLimitPercentClamped (quota/property/percent-clamped).
func propertyNewLimitPercentClamped(t *testing.T, seed uint64) {
	t.Logf("design seed %d (rapid v1.2 takes its own seed from -rapid.seed)", seed)
	rapid.Check(t, func(t *rapid.T) {
		l := limitGen(KindWeekly).Draw(t, "limit")
		if l.Percent < 0 || l.Percent > 100 {
			t.Fatalf("Percent = %d", l.Percent)
		}
	})
}

// propertyLimitElapsedBounded (quota/property/elapsed-bounded).
func propertyLimitElapsedBounded(t *testing.T, seed uint64) {
	t.Logf("design seed %d (rapid v1.2 takes its own seed from -rapid.seed)", seed)
	rapid.Check(t, func(t *rapid.T) {
		l := limitGen(KindSession).Draw(t, "limit")
		now := nowGen().Draw(t, "now")
		if e := l.Elapsed(now); e < 0 || e > 1 {
			t.Fatalf("Elapsed = %v", e)
		}
		if c := l.CursorPosition(now); c < 0 || c > 100 {
			t.Fatalf("CursorPosition = %d", c)
		}
		if r := l.Remaining(now); r < 0 {
			t.Fatalf("Remaining = %v", r)
		}
		if l.IsOnTrack(now) != (l.Pace(now) <= 0) {
			t.Fatal("IsOnTrack disagrees with Pace")
		}
	})
}

// propertyLimitExhaustsInBeforeReset (quota/property/exhaustion-before-reset).
func propertyLimitExhaustsInBeforeReset(t *testing.T, seed uint64) {
	t.Logf("design seed %d (rapid v1.2 takes its own seed from -rapid.seed)", seed)
	rapid.Check(t, func(t *rapid.T) {
		l := limitGen(KindWeekly).Draw(t, "limit")
		now := nowGen().Draw(t, "now")
		left, ok := l.ExhaustsIn(now)
		if ok && (left < 0 || left >= l.Remaining(now)) {
			t.Fatalf("ExhaustsIn = %v, Remaining = %v", left, l.Remaining(now))
		}
		if !ok && left != 0 {
			t.Fatalf("no exhaustion but %v", left)
		}
	})
}

// propertyNewProgressBounded (quota/property/progress-bounded).
func propertyNewProgressBounded(t *testing.T, seed uint64) {
	t.Logf("design seed %d (rapid v1.2 takes its own seed from -rapid.seed)", seed)
	rapid.Check(t, func(t *rapid.T) {
		tokens := rapid.IntRange(-1_000_000, 10_000_000).Draw(t, "tokens")
		size := rapid.IntRange(-10, 2_000_000).Draw(t, "size")
		p := NewProgress(tokens, size)
		if p.Percent < 0 || p.Percent > 100 {
			t.Fatalf("NewProgress(%d, %d) = %d", tokens, size, p.Percent)
		}
		a := rapid.IntRange(0, 100).Draw(t, "a")
		b := rapid.IntRange(a, 100).Draw(t, "b")
		if (Progress{Percent: a}).Level() > (Progress{Percent: b}).Level() {
			t.Fatalf("Level not monotonic between %d and %d", a, b)
		}
	})
}

// TestTimestampTotal: UnmarshalJSON never fails, whatever the bytes.
func TestTimestampTotal(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		raw := rapid.SliceOf(rapid.Byte()).Draw(t, "raw")
		var ts Timestamp
		if err := ts.UnmarshalJSON(raw); err != nil {
			t.Fatalf("UnmarshalJSON(%q) = %v", raw, err)
		}
		// And inside a document, a field never voids its siblings
		var doc struct {
			At Timestamp `json:"at"`
			N  int       `json:"n"`
		}
		n := rapid.IntRange(0, 1000).Draw(t, "n")
		epochSecs := rapid.Int64Range(-10, 5_000_000_000).Draw(t, "epoch")
		body, _ := json.Marshal(map[string]any{"at": epochSecs, "n": n})
		if err := json.Unmarshal(body, &doc); err != nil || doc.N != n {
			t.Fatalf("decode %s = %+v, %v", body, doc, err)
		}
		if epochSecs <= 0 && !doc.At.Time.IsZero() {
			t.Fatalf("a non-positive epoch decoded to %v", doc.At)
		}
	})
}

func setGen() *rapid.Generator[Set] {
	return rapid.Custom(func(t *rapid.T) Set {
		return Set{
			Context: limitGen(KindContext).Draw(t, "context"),
			Session: limitGen(KindSession).Draw(t, "session"),
			Weekly:  limitGen(KindWeekly).Draw(t, "weekly"),
			Scoped:  rapid.SliceOfN(limitGen(KindScoped), 0, 3).Draw(t, "scoped"),
		}
	})
}

// propertyResolveStdinWins (quota/property/resolve-stdin-wins).
func propertyResolveStdinWins(t *testing.T, seed uint64) {
	t.Logf("design seed %d (rapid v1.2 takes its own seed from -rapid.seed)", seed)
	rapid.Check(t, func(t *rapid.T) {
		stdin, api := setGen().Draw(t, "stdin"), setGen().Draw(t, "api")
		got := Resolve(stdin, api)
		if stdin.Session.IsValid() && got.Session != stdin.Session {
			t.Fatal("a valid stdin session was replaced")
		}
		if stdin.Weekly.IsValid() && got.Weekly != stdin.Weekly {
			t.Fatal("a valid stdin weekly was replaced")
		}
		if got.Context != stdin.Context {
			t.Fatal("the context window came from the API")
		}
	})
}

// propertyResolveNeverInvents (quota/property/resolve-never-invents).
func propertyResolveNeverInvents(t *testing.T, seed uint64) {
	t.Logf("design seed %d (rapid v1.2 takes its own seed from -rapid.seed)", seed)
	rapid.Check(t, func(t *rapid.T) {
		stdin, api := setGen().Draw(t, "stdin"), setGen().Draw(t, "api")
		got := Resolve(stdin, api)
		if !stdin.Session.IsValid() && !api.Session.IsValid() && got.Session.IsValid() {
			t.Fatal("a session quota absent from both appeared")
		}
		if !stdin.Weekly.IsValid() && !api.Weekly.IsValid() && got.Weekly.IsValid() {
			t.Fatal("a weekly quota absent from both appeared")
		}
		if len(got.Scoped) != len(api.Scoped) {
			t.Fatal("scoped quotas do not come from the API alone")
		}
	})
}
