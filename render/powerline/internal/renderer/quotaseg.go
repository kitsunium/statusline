// Package renderer provides status line rendering.
package renderer

import (
	"strconv"
	"strings"
	"time"

	"github.com/kitsunium/statusline/render/powerline/internal/model"
)

// Quota segment constants.
const (
	// segBarWidth is the bar width of a quota segment on the shared line.
	segBarWidth int = 10
)

// quotaSegment is one quota rendered as a chained powerline segment rather
// than as an isolated pill, so the whole line reads as a single ribbon the way
// the original status line did.
type quotaSegment struct {
	limit model.Limit
	bg    string
	cap   string
	ink   string
	// now is the instant the countdown and the burn cursor are drawn at
	now time.Time
}

// quotaSegments builds the chain of quota segments for the shared line.
// Colours are fixed per bucket rather than derived from state: the ribbon is an
// identity, and a segment that changes hue as consumption moves would make the
// line flicker between redraws.
//
// Params:
//   - data: everything the status line renders
//
// Returns:
//   - []quotaSegment: quotas to chain, most local first
func quotaSegments(data model.StatusLineData) []quotaSegment {
	limits := data.Limits
	segments := make([]quotaSegment, 0, len(limits.Scoped)+3)

	// Lead with the context window: it is the quota that bites first, and the
	// only one that is always present
	if limits.Context.IsValid() && !isKindHidden(model.KindContext) {
		segments = append(segments, quotaSegment{
			limit: limits.Context,
			bg:    BgContext,
			cap:   FgContext,
			ink:   FgContextInk,
			now:   data.Now,
		})
	}

	return segments
}

// modelQuotas returns the account quotas that belong beside the model name.
//
// The session and weekly windows, and any quota scoped to the model in use,
// are all limits on the same thing: what this account may spend on this model.
// They read as one group, so they are drawn as one.
//
// Params:
//   - data: everything the status line renders
//
// Returns:
//   - []model.Limit: quotas to draw inside the model segment
func modelQuotas(data model.StatusLineData) []model.Limit {
	limits := data.Limits
	quotas := make([]model.Limit, 0, 3)

	// The rolling five-hour window
	if limits.Session.IsValid() && !isKindHidden(model.KindSession) {
		quotas = append(quotas, limits.Session)
	}
	// The plan-wide seven-day window
	if limits.Weekly.IsValid() && !isKindHidden(model.KindWeekly) {
		quotas = append(quotas, limits.Weekly)
	}
	// Only the scoped quotas that restrict the model actually in use
	if !isKindHidden(model.KindScoped) {
		quotas = append(quotas, limits.ScopedFor(data.Model.FullName())...)
	}

	return quotas
}

// segmentColors returns the fixed colours of a quota segment.
//
// Params:
//   - limit: quota to colour
//
// Returns:
//   - bg: segment background
//   - cap: segment background as a foreground, for the separator
//   - ink: segment text colour
func segmentColors(limit model.Limit) (bg, cap, ink string) {
	// Map each bucket onto the palette the project already uses
	switch limit.Kind {
	// The rolling five-hour quota
	case model.KindSession:
		return BgHueSession, FgHueSession, FgHueSessionInk
	// The plan-wide seven-day quota keeps the grey the original line used
	case model.KindWeekly:
		return BgWeekly, FgWeekly, FgWeeklyText
	// A model-scoped quota takes the hue of the model it restricts
	case model.KindScoped:
		return GetModelColors(limit.Label)
	// Anything else falls back to the neutral ground
	default:
		return BgQuota, FgQuota, FgQuotaText
	}
}

// renderQuotaSegment writes one quota as a chained powerline segment.
//
// Params:
//   - sb: string builder to write to
//   - seg: quota segment to render
//   - nextBg: background of the segment that follows
//   - fit: what the line gives up to fit the terminal
func renderQuotaSegment(sb *strings.Builder, seg quotaSegment, nextBg string, fit lineFit) {
	limit := seg.limit
	// Without its bar, a quota is its icon and its figure
	if fit.dropBar(limit.Kind) {
		sb.WriteString(seg.bg + seg.ink + Bold + " " + compactLabel(limit) + " " + strconv.Itoa(limit.Percent) + "%" + Reset)
	} else {
		cursor := noCursor
		// Place the even-burn cursor only when the window makes it meaningful
		if limit.HasWindow() {
			cursor = limit.CursorPosition(seg.now)
		}
		bar := RenderProgressBarWidth(limit.Progress(), cursor, segBarWidth, seg.ink, seg.bg+seg.ink+Bold)

		// Write the label, the bar and the consumed percentage
		sb.WriteString(seg.bg + seg.ink + Bold + " " + QuotaLabel(limit) + " " + Reset)
		sb.WriteString(seg.bg + seg.ink + Bold + bar + Reset)
		sb.WriteString(seg.bg + seg.ink + Bold + " " + strconv.Itoa(limit.Percent) + "%" + Reset)
	}
	// Append the countdown to the refill, which is what the bar cannot say
	if limit.HasWindow() && !fit.dropCountdown(limit.Kind) {
		sb.WriteString(seg.bg + seg.ink + " " + glyphs.Reset + FormatDuration(limit.Remaining(seg.now)) + Reset)
	}
	sb.WriteString(seg.bg + " " + Reset)

	// Write the separator into whatever follows
	sb.WriteString(nextBg + seg.cap + SepRight + Reset)
}

// compactLabel names a chained quota that lost its bar: the context window
// keeps only its icon, which the figure beside it explains; without a glyph
// set the name stays, or the figure would stand unexplained.
//
// Params:
//   - limit: quota to label
//
// Returns:
//   - string: icon alone, or the full label
func compactLabel(limit model.Limit) string {
	// Only the context window has an icon that says enough on its own
	if limit.Kind == model.KindContext && glyphs.Ctx != "" {
		return glyphs.Ctx
	}
	return QuotaLabel(limit)
}
