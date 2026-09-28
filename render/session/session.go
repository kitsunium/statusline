package session

import (
	"encoding/json"
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

// parse never fails: the renderer degrades to what it can establish alone,
// which beats a raw error on every redraw. Invalid JSON decodes nothing; a
// mistyped field leaves what encoding/json decoded around it, exactly as the
// legacy product did.
func parse(data []byte) Payload {
	var p Payload
	_ = json.Unmarshal(data, &p)
	return p
}

func (p *Payload) modelInfo() ModelInfo {
	name := p.Model.DisplayName
	if name == "" {
		name = defaultModelName
	}
	base, version := splitModelName(name)
	return ModelInfo{Name: base, Version: version}
}

// splitModelName cuts at the first space followed by something.
func splitModelName(name string) (string, string) {
	for idx, ch := range name {
		if ch == ' ' && idx+1 < len(name) {
			return name[:idx], name[idx+1:]
		}
	}
	return name, ""
}

func (p *Payload) workingDir() string {
	if p.Workspace.CurrentDir == "" {
		return defaultWorkingDir
	}
	return p.Workspace.CurrentDir
}

func (p *Payload) contextWindowSize() int {
	if p.ContextWindow.ContextWindowSize == 0 {
		return defaultContextWindowSize
	}
	return p.ContextWindow.ContextWindowSize
}

func (p *Payload) totalTokens() int {
	return p.ContextWindow.TotalInputTokens + p.ContextWindow.TotalOutputTokens
}

// progress prefers the host's own percentage: cumulative token totals
// exceed the window after a compaction.
func (p *Payload) progress() quota.Progress {
	if p.ContextWindow.UsedPercentage != nil {
		return quota.Progress{Percent: min(int(*p.ContextWindow.UsedPercentage), 100)}
	}
	return quota.NewProgress(p.totalTokens(), p.contextWindowSize())
}

// stdinLimits keeps absent buckets absent: a plan without a weekly cap
// yields a set with only the session limit, never a 0 % bar.
func (p *Payload) stdinLimits() quota.Set {
	set := quota.Set{
		Context: quota.NewLimit(quota.KindContext, "ctx", p.Progress().Percent, time.Time{}, 0, quota.SourceStdin),
	}
	if limit, ok := p.RateLimits.FiveHour.limit(quota.KindSession, "session", quota.SessionWindow); ok {
		set.Session = limit
	}
	if limit, ok := p.RateLimits.SevenDay.limit(quota.KindWeekly, "weekly", quota.WeeklyWindow); ok {
		set.Weekly = limit
	}
	return set
}

// percent prefers used_percentage, then utilization.
func (r *PayloadRateLimit) percent() (int, bool) {
	if r == nil {
		return 0, false
	}
	if r.UsedPercentage != nil {
		return int(*r.UsedPercentage), true
	}
	if r.Utilization != nil {
		return int(*r.Utilization), true
	}
	return 0, false
}

// limit builds the bucket's Limit, false when it carries no percentage.
func (r *PayloadRateLimit) limit(kind quota.Kind, label string, window time.Duration) (quota.Limit, bool) {
	percent, ok := r.percent()
	if !ok {
		return quota.Limit{}, false
	}
	return quota.NewLimit(kind, label, percent, r.ResetsAt.Time, window, quota.SourceStdin), true
}

func (m ModelInfo) fullName() string {
	if m.Version == "" {
		return m.Name
	}
	return m.Name + " " + m.Version
}

// shortName drops parenthesised qualifiers such as "(1M context)".
func (m ModelInfo) shortName() string {
	if m.Version == "" {
		return m.Name
	}
	version := m.Version
	for idx, ch := range version {
		if ch == ' ' || ch == '(' {
			version = version[:idx]
			break
		}
	}
	if version == "" {
		return m.Name
	}
	return m.Name + " " + version
}

func effortRank(level string) (int, bool) {
	for idx, known := range effortScale {
		if known == level {
			return idx + 1, true
		}
	}
	return 0, false
}

func effortSteps() int { return len(effortScale) }
