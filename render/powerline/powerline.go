package powerline

import (
	"strconv"
	"strings"

	"github.com/kitsunium/statusline/render/powerline/internal/model"
	"github.com/kitsunium/statusline/render/powerline/internal/renderer"
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

// render converts the frame to the renderer's vocabulary and draws it.
func render(frame Frame) string {
	return renderer.NewPowerline().Render(toData(frame))
}

// condensed asks the renderer which segments gave way.
func condensed(frame Frame) []string {
	return renderer.Condensed(toData(frame))
}

func visibleWidth(s string) int {
	return renderer.VisibleWidth(s)
}

// parseWidth: the host gives the status line its room in COLUMNS; the
// process's own stdout is a pipe, and /dev/tty would report the whole window.
func parseWidth(value string) int {
	width, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || width <= 0 || width > maxWidth {
		return defaultWidth
	}
	return width
}

// iconsFromEnv: every icon is on unless its switch says false, 0 or no.
func iconsFromEnv(getenv func(string) string) Icons {
	icons := Icons{OS: true, Path: true, Git: true, Model: true}
	for name, dst := range map[string]*bool{
		"STATUSLINE_ICON_OS":    &icons.OS,
		"STATUSLINE_ICON_PATH":  &icons.Path,
		"STATUSLINE_ICON_GIT":   &icons.Git,
		"STATUSLINE_ICON_MODEL": &icons.Model,
	} {
		if val := getenv(name); val != "" {
			*dst = parseBool(val)
		}
	}
	return icons
}

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
