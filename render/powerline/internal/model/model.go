// Package model gives the renderer ported from kodflow/status-line the
// vocabulary it was written against, as aliases of the design's value
// objects. It lives in the powerline component's internal zone (D19).
package model

import (
	"time"

	"github.com/kitsunium/statusline/quota"
	"github.com/kitsunium/statusline/render/session"
	"github.com/kitsunium/statusline/snapshot"
)

// Quotas.
type (
	Limit         = quota.Limit
	LimitKind     = quota.Kind
	LimitSource   = quota.Source
	LimitSet      = quota.Set
	ExtraUsage    = quota.Extra
	Progress      = quota.Progress
	ProgressLevel = quota.Level
)

// Quota constants.
const (
	KindContext   = quota.KindContext
	KindSession   = quota.KindSession
	KindWeekly    = quota.KindWeekly
	KindScoped    = quota.KindScoped
	KindExtra     = quota.KindExtra
	SourceStdin   = quota.SourceStdin
	SourceAPI     = quota.SourceAPI
	SourceNone    = quota.SourceNone
	LevelLow      = quota.LevelLow
	LevelMedium   = quota.LevelMedium
	LevelHigh     = quota.LevelHigh
	LevelCritical = quota.LevelCritical
	SessionWindow = quota.SessionWindow
	WeeklyWindow  = quota.WeeklyWindow
)

// NewLimit builds a clamped limit.
func NewLimit(kind LimitKind, label string, percent int, resetsAt time.Time, window time.Duration, source LimitSource) Limit {
	return quota.NewLimit(kind, label, percent, resetsAt, window, source)
}

// NewProgress computes a usage percentage.
func NewProgress(totalTokens, contextSize int) Progress {
	return quota.NewProgress(totalTokens, contextSize)
}

// Collected state.
type (
	GitStatus     = snapshot.GitStatus
	CodeChanges   = snapshot.Changes
	MCPSource     = snapshot.MCPSource
	MCPServer     = snapshot.MCPServer
	MCPServers    = snapshot.MCPServers
	TaskItem      = snapshot.TaskItem
	TaskList      = snapshot.TaskList
	Epic          = snapshot.Epic
	TaskBoard     = snapshot.TaskBoard
	ServiceHealth = snapshot.Health
	OSType        = snapshot.OS
	SystemInfo    = snapshot.System
	UpdateInfo    = snapshot.UpdateNotice
)

// Collected state constants.
const (
	MCPSourceUnknown = snapshot.MCPSourceUnknown
	MCPSourceManaged = snapshot.MCPSourceManaged
	MCPSourceCLI     = snapshot.MCPSourceCLI
	MCPSourceLocal   = snapshot.MCPSourceLocal
	MCPSourceProject = snapshot.MCPSourceProject
	MCPSourceUser    = snapshot.MCPSourceUser
	MCPSourcePlugin  = snapshot.MCPSourcePlugin
	TaskPending      = snapshot.TaskPending
	TaskInProgress   = snapshot.TaskInProgress
	TaskWaiting      = snapshot.TaskWaiting
	TaskCompleted    = snapshot.TaskCompleted
	NoEpic           = snapshot.NoEpic
	HealthUnknown    = snapshot.HealthUnknown
	HealthOK         = snapshot.HealthOK
	HealthDegraded   = snapshot.HealthDegraded
	HealthDown       = snapshot.HealthDown
	OSLinux          = snapshot.OSLinux
	OSDarwin         = snapshot.OSDarwin
	OSWindows        = snapshot.OSWindows
	OSUnknown        = snapshot.OSUnknown
)

// NewCodeChanges builds a changes count.
func NewCodeChanges(added, removed int) CodeChanges {
	return CodeChanges{Added: added, Removed: removed}
}

// ToolKey normalises a name the way tool names spell it.
func ToolKey(name string) string { return snapshot.ToolKey(name) }

// The session.
type ModelInfo = session.ModelInfo

// Effort levels.
const (
	EffortLow    = session.EffortLow
	EffortMedium = session.EffortMedium
	EffortHigh   = session.EffortHigh
	EffortXHigh  = session.EffortXHigh
	EffortMax    = session.EffortMax
)

// EffortSteps returns the number of known effort levels.
func EffortSteps() int { return session.EffortSteps() }

// EffortRank places a level on the scale.
func EffortRank(level string) (int, bool) { return session.EffortRank(level) }

// IconConfig holds which icons are drawn.
type IconConfig struct {
	OS    bool
	Path  bool
	Git   bool
	Model bool
}

// DefaultIconConfig draws every icon.
func DefaultIconConfig() IconConfig {
	return IconConfig{OS: true, Path: true, Git: true, Model: true}
}

// TerminalInfo is the room the host gives the line.
type TerminalInfo struct {
	Width int
}

// StatusLineData is everything one render draws.
type StatusLineData struct {
	Model       ModelInfo
	Progress    Progress
	Limits      LimitSet
	Session     Limit
	Usage       Limit
	Icons       IconConfig
	Git         GitStatus
	System      SystemInfo
	Terminal    TerminalInfo
	Dir         string
	Time        string
	Cost        float64
	Effort      string
	FastMode    bool
	SessionName string
	Changes     CodeChanges
	MCP         MCPServers
	Update      UpdateInfo
	Health      ServiceHealth
	Tasks       TaskBoard
	Working     bool
	// Now is the instant countdowns, burn cursors and the task pulse are
	// drawn at; the legacy renderer read the wall clock itself.
	Now time.Time
}
