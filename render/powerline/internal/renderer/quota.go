// Package renderer provides status line rendering.
package renderer

import (
	"strings"

	"github.com/kitsunium/statusline/render/powerline/internal/model"
)

// quotaBarWidth is the bar width used inside a quota pill.
const quotaBarWidth int = 12

// Quota display names. An icon alone says what a pill is only to someone who
// already knows; the name beside it removes the guesswork. They are capitalised
// rather than upper-cased: a status line is glanced at, and a run of all-caps
// words reads slower than a capitalised one.
const (
	// nameContext names the conversation window.
	nameContext string = "Context"
	// nameSession names the rolling five-hour quota.
	nameSession string = "Session"
	// nameWeekly names the rolling seven-day quota.
	nameWeekly string = "Weekly"
)

// QuotaLabel renders the identity of a quota as an icon followed by its name.
//
// Params:
//   - limit: quota to label
//
// Returns:
//   - string: glyph, followed by the model name for a scoped quota
func QuotaLabel(limit model.Limit) string {
	// Map each kind onto its glyph and its name
	switch limit.Kind {
	// The conversation window, labelled by the fill level of its own glyph
	case model.KindContext:
		return Labelled(glyphs.Ctx, nameContext)
	// Every rate limit shares one gauge icon: they are the same kind of thing,
	// and the model chip belongs to the model segment on line one, not here
	case model.KindSession:
		return Labelled(glyphs.Quota, nameSession)
	// The plan-wide seven-day quota
	case model.KindWeekly:
		return Labelled(glyphs.Quota, nameWeekly)
	// A seven-day quota is named by the model family it restricts
	case model.KindScoped:
		return Labelled(glyphs.Quota, Capitalise(limit.Label))
	// Anything else falls back to whatever label it carries
	default:
		return Capitalise(limit.Label)
	}
}

// Labelled joins a glyph and a name, skipping the glyph when the active set
// has none. The text set spells identities out in words, so prefixing them
// with another word would only repeat what the name already says.
//
// Params:
//   - glyph: icon for the item, possibly empty
//   - name: display name
//
// Returns:
//   - string: glyph and name, or the name alone
func Labelled(glyph, name string) string {
	// An empty glyph leaves the name to stand on its own
	if glyph == "" {
		return name
	}
	return glyph + " " + name
}

// Capitalise upper-cases the first rune of a label, leaving the rest alone.
// Model names arrive from the API in whatever case it uses.
//
// Params:
//   - s: label to capitalise
//
// Returns:
//   - string: label with its first rune upper-cased
func Capitalise(s string) string {
	// An empty label has nothing to capitalise
	if s == "" {
		return s
	}
	runes := []rune(s)
	return strings.ToUpper(string(runes[0])) + string(runes[1:])
}
