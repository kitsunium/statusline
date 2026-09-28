// Package powerline renders the two-line Powerline status line.
//
// Exported API of design/domains/render.yaml (render/component/powerline).
// This file stands in for the shells kit generates (api_gen.go) until
// `kit gen` is available: every exported symbol only delegates to its
// unexported twin. The renderer itself lives in the internal zone.
package powerline

import (
	"time"

	"github.com/kitsunium/statusline/quota"
	"github.com/kitsunium/statusline/render/session"
	"github.com/kitsunium/statusline/snapshot"
)

// Icons says which icons are drawn.
type Icons struct {
	OS    bool
	Path  bool
	Git   bool
	Model bool
}

// Frame is everything one render draws.
type Frame struct {
	Model       session.ModelInfo
	Effort      string
	FastMode    bool
	SessionName string
	Cost        float64
	Limits      quota.Set
	Icons       Icons
	Width       int
	Dir         string
	Git         snapshot.GitStatus
	Changes     snapshot.Changes
	System      snapshot.System
	MCP         snapshot.MCPServers
	Health      snapshot.Health
	Tasks       snapshot.TaskBoard
	Working     bool
	Update      snapshot.UpdateNotice
	Now         time.Time
}

// Render draws the two lines.
func Render(frame Frame) string { return render(frame) }

// Condensed names the segments line one gave up to fit, and how far.
func Condensed(frame Frame) []string { return condensed(frame) }

// VisibleWidth counts the terminal cells a string occupies.
func VisibleWidth(s string) int { return visibleWidth(s) }

// ParseWidth reads a COLUMNS value; 120 when absent or implausible.
func ParseWidth(value string) int { return parseWidth(value) }

// IconsFromEnv reads the STATUSLINE_ICON_* switches.
func IconsFromEnv(getenv func(string) string) Icons { return iconsFromEnv(getenv) }
