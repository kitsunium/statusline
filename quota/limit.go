package quota

import (
	"bytes"
	"encoding/json"
	"strconv"
	"time"
)

// Progress thresholds, in percent.
const (
	// thresholdMedium is where a consumption stops being low.
	thresholdMedium int = 50
	// thresholdHigh is where a consumption becomes high.
	thresholdHigh int = 75
	// thresholdCritical is where a consumption becomes critical.
	thresholdCritical int = 90
	// maxPercent is the ceiling of every percentage.
	maxPercent int = 100
)

// UnmarshalJSON decodes an epoch in seconds or an RFC 3339 string. Mixing
// the two silently used to yield a zero time, which reads as "quota absent";
// anything undecodable is still the zero instant, but never an error, so a
// single odd field cannot void a whole payload.
func (t *Timestamp) UnmarshalJSON(data []byte) error {
	t.Time = time.Time{}
	trimmed := bytes.TrimSpace(data)
	// A JSON null or nothing at all carries no instant
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil
	}
	// A quoted value is an RFC 3339 string
	if trimmed[0] == '"' {
		var raw string
		if json.Unmarshal(trimmed, &raw) != nil {
			return nil
		}
		if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
			t.Time = parsed
		}
		return nil
	}
	epoch, err := strconv.ParseFloat(string(trimmed), 64)
	// A non-positive or unparseable epoch carries no instant
	if err != nil || epoch <= 0 {
		return nil
	}
	t.Time = time.Unix(int64(epoch), 0)
	return nil
}

// isValid: the context window is timeless, so a source is enough; every
// other kind needs a reset instant to compute a pace.
func (l Limit) isValid() bool {
	if l.Source == SourceNone {
		return false
	}
	if l.Kind == KindContext {
		return true
	}
	return !l.ResetsAt.IsZero()
}

// hasWindow needs both a window and a reset instant.
func (l Limit) hasWindow() bool {
	return l.Window > 0 && !l.ResetsAt.IsZero()
}

// remaining clamps a reset instant already past to zero.
func (l Limit) remaining(now time.Time) time.Duration {
	if l.ResetsAt.IsZero() {
		return 0
	}
	return max(l.ResetsAt.Sub(now), 0)
}

// elapsed is 0 before the window started and 1 once it is spent.
func (l Limit) elapsed(now time.Time) float64 {
	if !l.hasWindow() {
		return 0
	}
	remaining := l.remaining(now)
	if remaining <= 0 {
		return 1
	}
	if remaining >= l.Window {
		return 0
	}
	return float64(l.Window-remaining) / float64(l.Window)
}

// cursorPosition is the break-even percentage for the elapsed fraction.
func (l Limit) cursorPosition(now time.Time) int {
	return int(l.elapsed(now) * float64(maxPercent))
}

// pace compares the consumption with the even-burn reference.
func (l Limit) pace(now time.Time) int {
	return l.Percent - l.cursorPosition(now)
}

// isOnTrack: at or under the reference is sustainable.
func (l Limit) isOnTrack(now time.Time) bool {
	return l.pace(now) <= 0
}

// projected is 0 while too early in the window to extrapolate.
func (l Limit) projected(now time.Time) int {
	elapsed := l.elapsed(now)
	if elapsed <= 0 {
		return 0
	}
	return int(float64(l.Percent) / elapsed)
}

// exhaustsIn derives the burn rate from the elapsed part of the window.
func (l Limit) exhaustsIn(now time.Time) (time.Duration, bool) {
	elapsed := l.elapsed(now)
	// No burn rate can be derived before the window starts
	if elapsed <= 0 || l.Percent <= 0 {
		return 0, false
	}
	spent := time.Duration(elapsed * float64(l.Window))
	perPercent := spent / time.Duration(l.Percent)
	left := time.Duration(maxPercent-l.Percent) * perPercent
	// Exhaustion beyond the reset is not a real exhaustion
	if left >= l.remaining(now) {
		return 0, false
	}
	return left, true
}

// progress reuses the shared severity thresholds.
func (l Limit) progress() Progress {
	return Progress{Percent: l.Percent}
}
