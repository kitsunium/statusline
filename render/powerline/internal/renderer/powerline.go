// Package renderer provides status line rendering.
package renderer

import (
	"strings"
	"time"

	"github.com/kitsunium/statusline/render/powerline/internal/model"
)

// Numeric constants for base conversion and progress.
const (
	// base10 is the decimal base for integer to string conversion.
	base10 int = 10
	// percentComplete represents 100% completion.
	percentComplete int = 100
)

// Powerline renders the status line in powerline style.
// It renders a status bar with segments and rounded corners.
type Powerline struct{}

// NewPowerline creates a new powerline renderer.
//
// Returns:
//   - *Powerline: new renderer instance
func NewPowerline() *Powerline {
	// Return empty struct as no state is needed
	return &Powerline{}
}

// Render generates the status line string.
//
// Params:
//   - data: all information needed for rendering
//
// Returns:
//   - string: formatted status line with ANSI codes
func (r *Powerline) Render(data model.StatusLineData) string {
	var sb strings.Builder

	r.renderCompact(&sb, data)

	// Return complete status line
	return sb.String()
}

// renderCompact renders the two-line shape: everything about the session on
// the first line, the ambient pills on the second.
//
// Params:
//   - sb: string builder to write to
//   - data: status line data
func (r *Powerline) renderCompact(sb *strings.Builder, data model.StatusLineData) {
	// Line one carries everything the session is: identity, MCP servers,
	// quotas, repository. Line two carries the epics, the MCP pill when it
	// is asked there, and an update notice.
	r.renderLine1(sb, data)
	sb.WriteString("\n" + LineGap())
	r.renderLine2With(sb, data, mcpOnLine2)
	sb.WriteString("\n")
}

// renderLine1 renders the first line, condensed to the terminal width.
//
// The line is drawn whole first; while it is wider than the terminal
// allows, it is drawn again one degradation step further (fitLevels). The
// MCP indicator inside the OS segment is part of every level, never given
// up. The second line is never condensed: a long task title wraps instead.
//
// Params:
//   - sb: string builder to write to
//   - data: status line data
func (r *Powerline) renderLine1(sb *strings.Builder, data model.StatusLineData) {
	line, _, _ := fitLine1(lineBudget(data.Terminal.Width), func(buf *strings.Builder, fit lineFit) {
		r.renderLine1Fit(buf, data, fit)
	})
	sb.WriteString(line)
}

// renderLine1Fit renders the first line with OS, Model, quota, Path, Git and
// Changes segments, giving up what fit says.
//
// Params:
//   - sb: string builder to write to
//   - data: status line data
//   - fit: what to leave out
func (r *Powerline) renderLine1Fit(sb *strings.Builder, data model.StatusLineData, fit lineFit) {
	// Get model background color for OS segment transition (use FullName for color detection)
	modelBg, _, _ := GetModelColors(data.Model.FullName())

	// Render OS segment (transitions to Model segment)
	var mcp model.MCPServers
	// The MCP indicator lives in the OS segment unless asked onto line two
	if !mcpOnLine2 {
		mcp = data.MCP
	}
	r.renderOSSegment(sb, data.System, data.Icons.OS, data.Health, mcp, data.Tasks.Unattributed, modelBg)

	// Build the quota chain first: each segment needs to know the colour of the
	// one that follows it to draw its separator
	segments := quotaSegments(data)

	// Whatever ends the quota chain hands over to the path, or straight to
	// the branch when the path gave way
	afterQuotas := BgBlue
	if fit.dropPath() && data.Git.IsInRepo() {
		afterQuotas = BgGit
	}

	// The model segment carries the account quotas that apply to this model
	modelNextBg := afterQuotas
	// Hand over to the first remaining segment when there is one
	if len(segments) > 0 {
		modelNextBg = segments[0].bg
	}
	modelData := &ModelSegmentData{
		Model:    data.Model,
		ShowIcon: data.Icons.Model,
		Progress: data.Progress,
		Cursor:   nil,
		NextBg:   modelNextBg,
		Effort:   data.Effort,
		FastMode: data.FastMode,
		Quotas:   modelQuotas(data),
		Fit:      fit,
		Now:      data.Now,
	}
	// The smallest levels leave the model its name alone
	modelData.ShowIcon = modelData.ShowIcon && !fit.dropModelIcon()
	r.renderModelSegment(sb, modelData)

	// Chain every quota, each handing over to the next and the last to the path
	for idx, seg := range segments {
		nextBg := afterQuotas
		// Hand over to the next quota when there is one
		if idx+1 < len(segments) {
			nextBg = segments[idx+1].bg
		}
		renderQuotaSegment(sb, seg, nextBg, fit)
	}

	r.renderRepoSegments(sb, data, fit)
}

// renderRepoSegments renders where the session works: path, git and
// changes, giving up what fit says.
//
// Params:
//   - sb: string builder to write to
//   - data: status line data
//   - fit: what to leave out
func (r *Powerline) renderRepoSegments(sb *strings.Builder, data model.StatusLineData, fit lineFit) {
	// Determine what follows git segment (or path if no git)
	changesNextBg := ""
	// Determine next segment background color based on changes
	if data.Changes.HasAdded() {
		// Use green background for added lines
		changesNextBg = BgGreen
		// Check if only removed changes exist
	} else if data.Changes.HasRemoved() {
		// Use red background for removed lines only
		changesNextBg = BgRed
	}

	// A line too narrow for the changes leaves them to the git counters
	changes := data.Changes
	if fit.dropChanges() {
		changes = model.CodeChanges{}
		changesNextBg = ""
	}

	// Render path segment (pass changesNextBg for case when no git); only
	// the tightest level gives it up, and only when the branch stays
	if !fit.dropPath() || !data.Git.IsInRepo() {
		r.renderPathSegment(sb, data.Dir, data.Git.IsInRepo(), data.Icons.Path, changesNextBg, fit.pathMax())
	}

	// Render git segment if in repo
	r.renderGitSegment(sb, data.Git, data.Icons.Git, changesNextBg, fit.branchMax())
	// Render code changes if any
	r.renderChangesSegment(sb, changes)
}

// renderLine2 renders the second line with the MCP pill on it.
//
// Params:
//   - sb: string builder to write to
//   - data: status line data
func (r *Powerline) renderLine2(sb *strings.Builder, data model.StatusLineData) {
	r.renderLine2With(sb, data, true)
}

// renderLine2With renders the second line with dynamic pills (epics, MCP,
// Update).
//
// Params:
//   - sb: string builder to write to
//   - data: status line data
//   - withMCP: whether the MCP pill goes on this line
func (r *Powerline) renderLine2With(sb *strings.Builder, data model.StatusLineData, withMCP bool) {
	// Track if we've rendered anything
	hasContent := false

	// The epics lead the line: they are the work in progress, the rest is
	// ambient. Nothing is shortened; a long title wraps rather than hides.
	for _, epic := range data.Tasks.Epics {
		r.renderEpicPill(sb, epic, epic.Active && data.Working, data.Now)
		hasContent = true
	}

	// One MCP pill sums up every server, after the epics, unless line one
	// already carries it
	if withMCP && len(data.MCP) > 0 {
		// Add separator space if previous content exists
		if hasContent {
			sb.WriteString(" ")
		}
		r.renderMCPPill(sb, data.MCP)
		hasContent = true
	}

	// Render update pill if update is available
	if data.Update.Available {
		// Add separator space if previous content exists
		if hasContent {
			sb.WriteString(" ")
		}
		r.renderUpdatePill(sb, data.Update)
	}
}

// noEpicLabel names the pill of the tasks filed under no epic.
const noEpicLabel string = "T\u00e2ches"

// renderEpicPill renders one epic as a mauve pill.
//
// Collapsed, the pill says what the epic is and how far it went: its title
// and done/total. Expanded — the active epic while the session works — it
// adds one cell per task and names the task that matters now.
//
// Params:
//   - sb: string builder to write to
//   - epic: open epic to draw
//   - expanded: whether to draw the cells and the headline
func (r *Powerline) renderEpicPill(sb *strings.Builder, epic model.Epic, expanded bool, now time.Time) {
	label := epic.Title
	// The tasks filed under no epic, and a nameless epic, still get a name
	if epic.ID == model.NoEpic {
		label = noEpicLabel
	} else if label == "" {
		label = "#" + itoa(epic.ID)
	}
	list := epic.Tasks
	sb.WriteString(" " + FgEpic + LeftRound + Reset)
	sb.WriteString(BgEpic + FgEpicInk + Bold + " " + label + " " + itoa(list.Done()) + "/" + itoa(list.Total()) + Reset)
	// Only the epic under way, only while it moves, shows its cells
	if expanded {
		renderEpicCells(sb, list, now)
		// Always name a task: the one under way, else the next one
		if subject, _ := list.Headline(); subject != "" {
			sb.WriteString(BgEpic + FgEpicInk + " " + subject + Reset)
		}
	}
	// The subagents started for this epic travel with it
	if epic.Subagents > 0 {
		sb.WriteString(BgEpic + FgEpicInk + Bold + " " + glyphs.Subagents + " " + itoa(epic.Subagents) + Reset)
	}
	sb.WriteString(BgEpic + " " + Reset + FgEpic + RightRound + Reset)
}

// renderEpicCells renders one cell per task, filled from the left: done, then
// under way, then everything else, whatever the ids — a task created late and
// finished early must not light a cell on the far right of an empty bar.
//
// Params:
//   - sb: string builder to write to
//   - list: tasks of the expanded epic
func renderEpicCells(sb *strings.Builder, list model.TaskList, now time.Time) {
	// An epic focused before its first task has no cell to draw
	if list.Total() == 0 {
		return
	}
	done, active := list.Done(), list.Active()
	sb.WriteString(BgEpic + " " + FgEpicInk + strings.Repeat(glyphs.TaskDone, done) + Reset)
	// The started cells pulse, one frame per second: the line is a still image
	// between redraws, and a refresh interval of one second is the fastest
	// cadence the host offers, so every other frame draws them brighter
	activeInk := FgEpicActive
	if pulseOn(now) {
		activeInk = FgEpicPulse
	}
	sb.WriteString(BgEpic + activeInk + strings.Repeat(glyphs.TaskDone, active) + Reset)
	// Only the task under way stands out; pending and waiting share the track
	sb.WriteString(BgEpic + FgEpicTrack + strings.Repeat(glyphs.TaskOpen, list.Total()-done-active) + Reset)
}

// pulseOn reports whether the current frame draws the started tasks bright.
//
// Returns:
//   - bool: true on even seconds
func pulseOn(now time.Time) bool {
	// Alternate on the wall clock so consecutive redraws differ
	return now.Unix()%2 == 0
}

// renderOSSegment renders the operating system segment.
//
// Params:
//   - sb: string builder to write to
//   - sys: system information
//   - showIcon: whether to show the OS icon
//   - health: state of Claude's services, drawn beside the icon when known
//   - mcp: MCP servers to sum up after the health glyph, nil for none
//   - subagents: running subagents tied to no epic on line 2
//   - nextBg: background color of the next segment
func (r *Powerline) renderOSSegment(sb *strings.Builder, sys model.SystemInfo, showIcon bool, health model.ServiceHealth, mcp model.MCPServers, subagents int, nextBg string) {
	// Write left rounded cap
	sb.WriteString(FgWhite + LeftRound + Reset)
	// Check if icon should be shown
	if showIcon {
		icon := GetOSIcon(sys.OS, sys.IsDocker)
		// Write icon with background
		sb.WriteString(BgWhite + FgBlack + Bold + " " + icon + " " + Reset)
	} else {
		// Write empty space without icon
		sb.WriteString(BgWhite + FgBlack + Bold + "  " + Reset)
	}
	// The health glyph sits inside the same white ground, coloured by state
	if color := healthColor(health); color != "" && !isHidden(hideHealth) {
		sb.WriteString(BgWhite + color + glyphs.Health + " " + Reset)
	}
	// The MCP servers the session can reach, between health and subagents
	writeMCPInline(sb, summarizeMCP(mcp))
	// Subagents working for no epic on show belong to the session as a whole
	if subagents > 0 {
		sb.WriteString(BgWhite + FgBlack + Bold + glyphs.Subagents + " " + itoa(subagents) + " " + Reset)
	}
	// Write separator to next segment
	sb.WriteString(nextBg + FgWhite + SepRight + Reset)
}

// healthColor returns the glyph colour for a service health level.
//
// Params:
//   - health: aggregate state of Claude's services
//
// Returns:
//   - string: foreground escape, empty when nothing should be drawn
func healthColor(health model.ServiceHealth) string {
	// Map each known level onto its colour
	switch health {
	// Every counted service is operational
	case model.HealthOK:
		return FgHealthOK
	// One service is degraded
	case model.HealthDegraded:
		return FgHealthDegraded
	// Two degraded services or one major outage
	case model.HealthDown:
		return FgHealthDown
	// An unknown state is not drawn: a light nobody checked would mislead
	default:
		return ""
	}
}

// renderModelSegment renders the AI model segment with integrated progress bar.
//
// Params:
//   - sb: string builder to write to
//   - data: model segment rendering data
func (r *Powerline) renderModelSegment(sb *strings.Builder, data *ModelSegmentData) {
	// Use FullName for color detection (includes version like "Opus 4.5")
	fullName := data.Model.FullName()
	bgColor, fgColor, textColor := GetModelColors(fullName)

	// Check if icon should be shown
	if data.ShowIcon {
		// Write model name with icon
		sb.WriteString(bgColor + textColor + Bold + " " + IconModel + " " + data.Model.ShortName() + Reset)
	} else {
		// Write model name without icon
		sb.WriteString(bgColor + textColor + Bold + " " + data.Model.ShortName() + Reset)
	}

	// Draw the reasoning effort right after the name it applies to
	if gauge := EffortGauge(data.Effort, textColor, GetModelTrack(fullName)); gauge != "" {
		sb.WriteString(bgColor + textColor + Bold + " " + gauge + Reset)
	}

	// Append the fast-mode marker when it is on
	if data.FastMode {
		sb.WriteString(bgColor + textColor + Bold + " " + glyphs.Fast + Reset)
	}

	// Draw the account quotas alongside the model they apply to. The session
	// window, the weekly window and any quota scoped to this model all limit
	// the same thing — what this account may spend on this model — so they
	// read as one group and are drawn as one, divided by a thin rule rather
	// than by a new segment.
	for idx, quota := range data.Quotas {
		// The first quota is the model's own session budget and runs straight
		// on from its name; the rest are divided by a thin rule and named
		if idx == 0 {
			sb.WriteString(bgColor + textColor + " " + Reset)
		} else {
			sb.WriteString(bgColor + textColor + " " + glyphs.Divider + Reset)
			sb.WriteString(bgColor + textColor + Bold + " " + data.Fit.quotaLabel(quota) + " " + Reset)
		}
		// The bar goes when the line is too narrow; the figure stays
		if data.Fit.dropBar(quota.Kind) {
			sb.WriteString(bgColor + textColor + Bold + itoa(quota.Percent) + "%" + Reset)
		} else {
			cursor := noCursor
			// Place the even-burn cursor only when the window makes it meaningful
			if quota.HasWindow() {
				cursor = quota.CursorPosition(data.Now)
			}
			bar := RenderProgressBarWidth(quota.Progress(), cursor, segBarWidth, textColor, bgColor+textColor)
			sb.WriteString(bgColor + textColor + bar + Bold + " " + itoa(quota.Percent) + "%" + Reset)
		}
		// Append the countdown to the refill, which the bar cannot say
		if quota.HasWindow() && !data.Fit.dropCountdown(quota.Kind) {
			sb.WriteString(bgColor + textColor + " " + glyphs.Reset + FormatDuration(quota.Remaining(data.Now)) + Reset)
		}
	}

	sb.WriteString(bgColor + " " + Reset)

	// Write separator to next segment
	sb.WriteString(data.NextBg + fgColor + SepRight + Reset)
}

// renderPathSegment renders the current directory segment.
//
// Params:
//   - sb: string builder to write to
//   - dir: directory path
//   - hasGit: whether git segment follows
//   - showIcon: whether to show the folder icon
//   - nextBg: background color of next segment if no git
//   - maxPath: path budget, 0 for the default one
func (r *Powerline) renderPathSegment(sb *strings.Builder, dir string, hasGit bool, showIcon bool, nextBg string, maxPath int) {
	truncated := TruncatePath(dir, maxPath)
	// Check if icon should be shown
	if showIcon {
		// Write path with folder icon and dark blue text
		sb.WriteString(BgBlue + FgBlueDark + Bold + " " + IconFolder + " " + truncated + " " + Reset)
	} else {
		// Write path without icon with dark blue text
		sb.WriteString(BgBlue + FgBlueDark + Bold + " " + truncated + " " + Reset)
	}

	// Determine separator style based on next segment
	if hasGit {
		// Write separator to git segment
		sb.WriteString(BgGit + FgBlue + SepRight + Reset)
		// Check if next segment has a colored background
	} else if nextBg != "" {
		// Write separator to next colored segment
		sb.WriteString(nextBg + FgBlue + SepRight + Reset)
		// No following segment
	} else {
		// Write final separator
		sb.WriteString(FgBlue + SepRight + Reset)
	}
}

// renderGitSegment renders the git branch and status segment.
//
// Params:
//   - sb: string builder to write to
//   - git: git status information
//   - showIcon: whether to show the git branch icon
//   - nextBg: background color of next segment for separator
//   - maxBranch: branch budget in runes, 0 for no limit
func (r *Powerline) renderGitSegment(sb *strings.Builder, git model.GitStatus, showIcon bool, nextBg string, maxBranch int) {
	// Skip if not in a git repository
	if !git.IsInRepo() {
		// Return early if not in repo
		return
	}

	// Check if icon should be shown
	if showIcon {
		// Write branch with icon and dark cyan text
		sb.WriteString(BgGit + FgGitInk + Bold + " " + IconGitBranch + " " + TruncateBranch(git.Branch, maxBranch))
	} else {
		// Write branch without icon with dark cyan text
		sb.WriteString(BgGit + FgGitInk + Bold + " " + TruncateBranch(git.Branch, maxBranch))
	}

	// Add modified indicator if present
	if git.Modified > 0 {
		sb.WriteString(" !" + itoa(git.Modified))
	}
	// Add untracked indicator if present
	if git.Untracked > 0 {
		sb.WriteString(" ?" + itoa(git.Untracked))
	}
	// Add the linked worktrees, where parallel work is going on
	if git.Worktrees > 0 {
		sb.WriteString(" " + glyphs.Worktree + " " + itoa(git.Worktrees))
	}

	// Write segment end with appropriate separator
	sb.WriteString(" " + Reset)
	// Check if next segment has background
	if nextBg != "" {
		// Write separator to next colored segment
		sb.WriteString(nextBg + FgGit + SepRight + Reset)
	} else {
		// Write final separator
		sb.WriteString(FgGit + SepRight + Reset)
	}
}

// renderChangesSegment renders the lines added/removed as powerline segments.
//
// Params:
//   - sb: string builder to write to
//   - changes: code changes information
func (r *Powerline) renderChangesSegment(sb *strings.Builder, changes model.CodeChanges) {
	// Skip if no changes
	if !changes.HasChanges() {
		// Return early if nothing to show
		return
	}

	// Render added segment if any
	if changes.HasAdded() {
		// Write added segment with dark green text on pale green background
		sb.WriteString(BgGreen + FgGreenText + Bold + " +" + itoa(changes.Added) + " " + Reset)

		// Determine separator destination
		if changes.HasRemoved() {
			// Separator to red segment
			sb.WriteString(BgRed + FgGreenSep + SepRight + Reset)
		} else {
			// Final separator
			sb.WriteString(FgGreenSep + SepRight + Reset)
		}
	}

	// Render removed segment if any
	if changes.HasRemoved() {
		// Write removed segment with dark red text on pale red background
		sb.WriteString(BgRed + FgRedText + Bold + " -" + itoa(changes.Removed) + " " + Reset)
		// Final separator
		sb.WriteString(FgRedSep + SepRight + Reset)
	}
}

// renderWeeklySegment renders the weekly API usage segment with burn-rate cursor.
// Auto-hidden when Usage.IsValid() is false (no API credentials).
//
// Params:
//   - sb: string builder to write to
//   - usage: weekly usage data
func (r *Powerline) renderWeeklySegment(sb *strings.Builder, usage model.Limit, now time.Time) {
	progress := usage.Progress()
	bar := RenderProgressBarWithCursor(progress, usage.CursorPosition(now), FgWeeklyText, BgWeekly+FgWeeklyText+Bold)

	// Write segment content
	sb.WriteString(BgWeekly + FgWeeklyText + Bold + " " + IconWeekly + " " + bar + " " + itoa(progress.Percent) + "% " + Reset)

	// Write separator to path segment
	sb.WriteString(BgBlue + FgWeekly + SepRight + Reset)
}

// renderUpdatePill renders the update notification pill.
//
// Params:
//   - sb: string builder to write to
//   - update: update information
func (r *Powerline) renderUpdatePill(sb *strings.Builder, update model.UpdateInfo) {
	// Skip if no update available
	if !update.Available {
		// Return early if nothing to show
		return
	}

	// Add space before pill
	sb.WriteString(" ")

	// Write left rounded cap (white)
	sb.WriteString(FgWhite + LeftRound + Reset)
	// Write update icon and version on white background
	sb.WriteString(BgWhite + FgBlack + Bold + " " + IconUpdate + " " + update.Version + " " + Reset)
	// Write right rounded cap
	sb.WriteString(FgWhite + RightRound + Reset)
}

// itoa converts an integer to string.
//
// Params:
//   - n: integer to convert
//
// Returns:
//   - string: string representation of the integer
func itoa(n int) string {
	// Handle zero case
	if n == 0 {
		// Return zero string
		return "0"
	}
	var digits []byte
	// Extract digits in reverse order
	for n > 0 {
		digits = append([]byte{byte('0' + n%base10)}, digits...)
		n /= base10
	}
	// Return converted string
	return string(digits)
}
