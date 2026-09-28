package session

import (
	"time"

	"github.com/kitsunium/statusline/quota"
)

// Fallbacks for absent fields.
const (
	// defaultModelName names a model the payload does not.
	defaultModelName string = "Claude"
	// defaultWorkingDir stands for a directory the payload does not give.
	defaultWorkingDir string = "~"
	// defaultContextWindowSize is the window of a payload without one.
	defaultContextWindowSize int = 200000
)

// effortScale is pinned here: the host only reports the current level's
// name, never the scale, so a level missing from it is reported as unknown
// rather than drawn on a wrong gauge.
var effortScale = []string{EffortLow, EffortMedium, EffortHigh, EffortXHigh, EffortMax}

// splitModelName cuts at the first space followed by something.
func splitModelName(name string) (string, string) {
	for idx, ch := range name {
		if ch == ' ' && idx+1 < len(name) {
			return name[:idx], name[idx+1:]
		}
	}
	return name, ""
}

// limit builds the bucket's Limit, false when it carries no percentage.
func (r *PayloadRateLimit) limit(kind string, label string, window time.Duration) (quota.Limit, bool) {
	percent, ok := r.percent()
	if !ok {
		return quota.Limit{}, false
	}
	return quota.NewLimit(kind, label, percent, r.ResetsAt.Time, window, quota.SourceStdin), true
}
