// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package session

import (
	"time"

	"github.com/kitsunium/statusline/quota"
)

func (p *Payload) contextWindowSize() int {
	if p.ContextWindow.ContextWindowSize == 0 {
		return defaultContextWindowSize
	}
	return p.ContextWindow.ContextWindowSize
}

func (p *Payload) modelInfo() ModelInfo {
	name := p.Model.DisplayName
	if name == "" {
		name = defaultModelName
	}
	base, version := splitModelName(name)
	return ModelInfo{Name: base, Version: version}
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

func (p *Payload) totalTokens() int {
	return p.ContextWindow.TotalInputTokens + p.ContextWindow.TotalOutputTokens
}

func (p *Payload) workingDir() string {
	if p.Workspace.CurrentDir == "" {
		return defaultWorkingDir
	}
	return p.Workspace.CurrentDir
}
