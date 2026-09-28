// Package quota holds the context window and the account's rate limits.
//
// Exported API of design/domains/quota.yaml. This file stands in for the
// shells kit generates (api_gen.go) until `kit gen` is available: every
// exported symbol here only delegates to its unexported twin.
package quota

import "time"

// SessionWindow is the rolling 5-hour rate limit window.
const SessionWindow time.Duration = 5 * time.Hour

// WeeklyWindow is the rolling 7-day rate limit window.
const WeeklyWindow time.Duration = 7 * 24 * time.Hour

// Kind identifies which quota a Limit measures.
type Kind string

// Limit kinds, from the most local to the most global quota.
const (
	KindContext Kind = "context"
	KindSession Kind = "session"
	KindWeekly  Kind = "weekly"
	KindScoped  Kind = "weekly_scoped"
	KindExtra   Kind = "extra"
)

// Source records where a Limit was read from.
type Source string

// Limit sources, by decreasing authority.
const (
	SourceStdin Source = "stdin"
	SourceAPI   Source = "api"
	SourceNone  Source = ""
)

// Level is the severity of a consumption.
type Level int

// Severity levels, from safe to critical.
const (
	LevelLow Level = iota
	LevelMedium
	LevelHigh
	LevelCritical
)

// Progress is a consumption percentage.
type Progress struct {
	Percent int `json:"percent"`
}

// NewProgress computes the context usage from token counts.
func NewProgress(totalTokens, contextSize int) Progress {
	return newProgress(totalTokens, contextSize)
}

// Level returns the severity of the consumption.
func (p Progress) Level() Level { return p.level() }

// Timestamp decodes an instant written as a Unix epoch in seconds or as an
// RFC 3339 string; anything else is the zero instant, never an error.
type Timestamp struct {
	time.Time
}

// UnmarshalJSON decodes both representations.
func (t *Timestamp) UnmarshalJSON(data []byte) error { return t.unmarshalJSON(data) }

// Limit is one quota and its refill schedule. A zero Limit means the plan
// does not expose that quota — never render it as 0 %.
type Limit struct {
	Kind     Kind          `json:"kind,omitempty"`
	Label    string        `json:"label,omitempty"`
	Percent  int           `json:"percent,omitempty"`
	ResetsAt time.Time     `json:"resets_at,omitzero"`
	Window   time.Duration `json:"window,omitempty"`
	Active   bool          `json:"active,omitempty"`
	Source   Source        `json:"source,omitempty"`
}

// NewLimit builds a Limit with its percentage clamped to 0-100.
func NewLimit(kind Kind, label string, percent int, resetsAt time.Time, window time.Duration, source Source) Limit {
	return newLimit(kind, label, percent, resetsAt, window, source)
}

// IsValid reports whether the limit carries usable data.
func (l Limit) IsValid() bool { return l.isValid() }

// HasWindow reports whether pace arithmetic is meaningful.
func (l Limit) HasWindow() bool { return l.hasWindow() }

// Remaining returns the time left before the quota refills, never negative.
func (l Limit) Remaining(now time.Time) time.Duration { return l.remaining(now) }

// Elapsed returns how far into the window now is, as a 0-1 fraction.
func (l Limit) Elapsed(now time.Time) float64 { return l.elapsed(now) }

// CursorPosition returns the consumption an even burn would show now.
func (l Limit) CursorPosition(now time.Time) int { return l.cursorPosition(now) }

// Pace returns how far ahead (positive) or behind an even burn the
// consumption is, in percentage points.
func (l Limit) Pace(now time.Time) int { return l.pace(now) }

// IsOnTrack reports whether the current burn rate lasts until the reset.
func (l Limit) IsOnTrack(now time.Time) bool { return l.isOnTrack(now) }

// Projected extrapolates the consumption at the end of the window.
func (l Limit) Projected(now time.Time) int { return l.projected(now) }

// ExhaustsIn returns the time left before exhaustion at the current rate,
// and false when the quota outlasts its window.
func (l Limit) ExhaustsIn(now time.Time) (time.Duration, bool) { return l.exhaustsIn(now) }

// Progress adapts the limit to the generic progress value.
func (l Limit) Progress() Progress { return l.progress() }

// Extra is the monthly extra-usage credit balance, in minor units.
type Extra struct {
	Enabled    bool   `json:"enabled,omitempty"`
	Percent    int    `json:"percent,omitempty"`
	UsedMinor  int    `json:"used_minor,omitempty"`
	LimitMinor int    `json:"limit_minor,omitempty"`
	Currency   string `json:"currency,omitempty"`
	Exponent   int    `json:"exponent,omitempty"`
	Source     Source `json:"source,omitempty"`
}

// IsValid reports whether the credit balance should be rendered.
func (e Extra) IsValid() bool { return e.isValid() }

// Set is every quota that applies to the session.
type Set struct {
	Context Limit   `json:"context"`
	Session Limit   `json:"session"`
	Weekly  Limit   `json:"weekly"`
	Scoped  []Limit `json:"scoped,omitempty"`
	Extra   Extra   `json:"extra"`
}

// Timed returns the time-boxed quotas that are present, most local first.
func (s Set) Timed() []Limit { return s.timed() }

// HasTimed reports whether any session, weekly or scoped quota is present.
func (s Set) HasTimed() bool { return s.hasTimed() }

// ScopedFor returns the scoped quotas matching the model in use.
func (s Set) ScopedFor(modelName string) []Limit { return s.scopedFor(modelName) }

// Resolve merges the quotas of stdin with the ones of the usage API.
func Resolve(stdin, api Set) Set { return resolve(stdin, api) }
