package powerline

import (
	"strings"

	"github.com/kitsunium/statusline/render/powerline/internal/model"
)

// Terminal width bounds.
const (
	// defaultWidth is assumed when COLUMNS is absent or invalid.
	defaultWidth int = 120
	// maxWidth bounds a plausible width; anything larger is a typo.
	maxWidth int = 10000
	// timeFormat is the clock the legacy data carried.
	timeFormat string = "15:04:05"
)

// parseBool is lenient: only an explicit negative switches off.
func parseBool(s string) bool {
	lower := strings.ToLower(strings.TrimSpace(s))
	return lower != "false" && lower != "0" && lower != "no"
}

// toData builds the legacy renderer's input: the context window is its own
// progress, and the session and weekly limits are duplicated where the
// legacy segments read them.
func toData(frame Frame) model.StatusLineData {
	return model.StatusLineData{
		Model:       frame.Model,
		Progress:    frame.Limits.Context.Progress(),
		Limits:      frame.Limits,
		Session:     frame.Limits.Session,
		Usage:       frame.Limits.Weekly,
		Icons:       model.IconConfig(frame.Icons),
		Git:         frame.Git,
		System:      frame.System,
		Terminal:    model.TerminalInfo{Width: frame.Width},
		Dir:         frame.Dir,
		Time:        frame.Now.Format(timeFormat),
		Cost:        frame.Cost,
		Effort:      frame.Effort,
		FastMode:    frame.FastMode,
		SessionName: frame.SessionName,
		Changes:     frame.Changes,
		MCP:         frame.MCP,
		Update:      frame.Update,
		Health:      frame.Health,
		Tasks:       frame.Tasks,
		Working:     frame.Working,
		Now:         frame.Now,
	}
}
