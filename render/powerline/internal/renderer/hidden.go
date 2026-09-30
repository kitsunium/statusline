// Package renderer provides status line rendering.
package renderer

import (
	"os"
	"strings"

	"github.com/kitsunium/statusline/render/powerline/internal/model"
)

// hideEnv names the environment variable listing the pills to hide.
const hideEnv string = "STATUSLINE_HIDE"

// Pill names accepted by STATUSLINE_HIDE.
const (
	// hideContext hides the context window pill.
	hideContext string = "context"
	// hideSession hides the five-hour quota pill.
	hideSession string = "session"
	// hideWeekly hides the seven-day quota pill.
	hideWeekly string = "weekly"
	// hideScoped hides the model-scoped quota pills.
	hideScoped string = "model"
	// hideHealth hides the service health glyph.
	hideHealth string = "health"
	// hideCredits hides the extra-usage credit pill.
	hideCredits string = "credits"
)

// hidden is the resolved set of pills to leave out, read once at startup.
var hidden = hiddenFromEnv()

// hiddenFromEnv parses the comma-separated list of pills to hide.
//
// Returns:
//   - map[string]bool: set of pill names to leave out
func hiddenFromEnv() map[string]bool {
	set := make(map[string]bool)
	raw := os.Getenv(hideEnv)
	// An unset variable hides nothing
	if raw == "" {
		return set
	}
	// Accept a forgiving list: spacing and case must not matter
	for _, name := range strings.Split(raw, ",") {
		trimmed := strings.ToLower(strings.TrimSpace(name))
		// Skip the empty entries a trailing comma leaves behind
		if trimmed == "" {
			continue
		}
		set[trimmed] = true
	}
	return set
}

// isHidden reports whether a named pill should be left out.
//
// Params:
//   - name: pill name as accepted by STATUSLINE_HIDE
//
// Returns:
//   - bool: true when the pill must not render
func isHidden(name string) bool {
	// Look the name up in the configured set
	return hidden[name]
}

// isKindHidden reports whether a quota kind should be left out.
//
// Params:
//   - kind: quota kind about to be rendered
//
// Returns:
//   - bool: true when the quota must not render
func isKindHidden(kind model.LimitKind) bool {
	// Map each kind onto the name the variable accepts
	switch kind {
	// The conversation window
	case model.KindContext:
		return isHidden(hideContext)
	// The rolling five-hour quota
	case model.KindSession:
		return isHidden(hideSession)
	// The plan-wide seven-day quota
	case model.KindWeekly:
		return isHidden(hideWeekly)
	// Every model-scoped quota at once
	case model.KindScoped:
		return isHidden(hideScoped)
	// The credit balance
	case model.KindExtra:
		return isHidden(hideCredits)
	// Anything else always renders
	default:
		return false
	}
}
