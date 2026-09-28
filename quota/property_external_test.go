package quota_test

import (
	"encoding/json"
	"testing"
	"time"

	"pgregory.net/rapid"

	"github.com/kitsunium/statusline/quota"
)

var epoch = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// limitGen draws any limit, valid or not, around epoch.
func limitGen(kind quota.Kind) *rapid.Generator[quota.Limit] {
	return rapid.Custom(func(t *rapid.T) quota.Limit {
		percent := rapid.IntRange(-500, 500).Draw(t, "percent")
		window := time.Duration(rapid.Int64Range(0, int64(8*24*time.Hour)).Draw(t, "window"))
		var reset time.Time
		if rapid.Bool().Draw(t, "hasReset") {
			reset = epoch.Add(time.Duration(rapid.Int64Range(-int64(10*24*time.Hour), int64(10*24*time.Hour)).Draw(t, "reset")))
		}
		source := rapid.SampledFrom([]quota.Source{quota.SourceStdin, quota.SourceAPI, quota.SourceNone}).Draw(t, "source")
		return quota.NewLimit(kind, "l", percent, reset, window, source)
	})
}

func nowGen() *rapid.Generator[time.Time] {
	return rapid.Custom(func(t *rapid.T) time.Time {
		return epoch.Add(time.Duration(rapid.Int64Range(-int64(10*24*time.Hour), int64(10*24*time.Hour)).Draw(t, "now")))
	})
}

// TestPropertyPercentClamped (quota/property/percent-clamped).
func TestPropertyPercentClamped(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		l := limitGen(quota.KindWeekly).Draw(t, "limit")
		if l.Percent < 0 || l.Percent > 100 {
			t.Fatalf("Percent = %d", l.Percent)
		}
	})
}

// TestPropertyElapsedBounded (quota/property/elapsed-bounded).
func TestPropertyElapsedBounded(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		l := limitGen(quota.KindSession).Draw(t, "limit")
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

// TestPropertyExhaustionBeforeReset (quota/property/exhaustion-before-reset).
func TestPropertyExhaustionBeforeReset(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		l := limitGen(quota.KindWeekly).Draw(t, "limit")
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

// TestPropertyProgressBounded (quota/property/progress-bounded).
func TestPropertyProgressBounded(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tokens := rapid.IntRange(-1_000_000, 10_000_000).Draw(t, "tokens")
		size := rapid.IntRange(-10, 2_000_000).Draw(t, "size")
		p := quota.NewProgress(tokens, size)
		if p.Percent < 0 || p.Percent > 100 {
			t.Fatalf("NewProgress(%d, %d) = %d", tokens, size, p.Percent)
		}
		a := rapid.IntRange(0, 100).Draw(t, "a")
		b := rapid.IntRange(a, 100).Draw(t, "b")
		if (quota.Progress{Percent: a}).Level() > (quota.Progress{Percent: b}).Level() {
			t.Fatalf("Level not monotonic between %d and %d", a, b)
		}
	})
}

// TestPropertyTimestampTotal (quota/property/timestamp-total).
func TestPropertyTimestampTotal(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		raw := rapid.SliceOf(rapid.Byte()).Draw(t, "raw")
		var ts quota.Timestamp
		if err := ts.UnmarshalJSON(raw); err != nil {
			t.Fatalf("UnmarshalJSON(%q) = %v", raw, err)
		}
		// And inside a document, a field never voids its siblings
		var doc struct {
			At quota.Timestamp `json:"at"`
			N  int             `json:"n"`
		}
		n := rapid.IntRange(0, 1000).Draw(t, "n")
		epochSecs := rapid.Int64Range(-10, 5_000_000_000).Draw(t, "epoch")
		body, _ := json.Marshal(map[string]any{"at": epochSecs, "n": n})
		if err := json.Unmarshal(body, &doc); err != nil || doc.N != n {
			t.Fatalf("decode %s = %+v, %v", body, doc, err)
		}
		if epochSecs <= 0 && !doc.At.IsZero() {
			t.Fatalf("a non-positive epoch decoded to %v", doc.At)
		}
	})
}

func setGen() *rapid.Generator[quota.Set] {
	return rapid.Custom(func(t *rapid.T) quota.Set {
		return quota.Set{
			Context: limitGen(quota.KindContext).Draw(t, "context"),
			Session: limitGen(quota.KindSession).Draw(t, "session"),
			Weekly:  limitGen(quota.KindWeekly).Draw(t, "weekly"),
			Scoped:  rapid.SliceOfN(limitGen(quota.KindScoped), 0, 3).Draw(t, "scoped"),
		}
	})
}

// TestPropertyResolveStdinWins (quota/property/resolve-stdin-wins).
func TestPropertyResolveStdinWins(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		stdin, api := setGen().Draw(t, "stdin"), setGen().Draw(t, "api")
		got := quota.Resolve(stdin, api)
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

// TestPropertyResolveNeverInvents (quota/property/resolve-never-invents).
func TestPropertyResolveNeverInvents(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		stdin, api := setGen().Draw(t, "stdin"), setGen().Draw(t, "api")
		got := quota.Resolve(stdin, api)
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
