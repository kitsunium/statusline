// Package renderer provides status line rendering.
package renderer

import (
	"os"
	"strconv"
	"strings"
)

// Vertical spacing between the two status line rows.
const (
	// lineGapEnv sets how many blank lines separate the two rows.
	lineGapEnv string = "STATUSLINE_LINE_GAP"
	// defaultLineGap keeps the rows adjacent; vertical breathing room is better
	// bought from the terminal's own cell height than from a line of status bar.
	defaultLineGap int = 0
	// maxLineGap bounds the spacing so the prompt stays usable.
	maxLineGap int = 3
)

// LineGap returns the blank lines written between the two status line rows.
//
// A terminal cell cannot be split, so vertical breathing room is whole blank
// lines or nothing.
//
// Returns:
//   - string: newlines written between rows
func LineGap() string {
	gap := defaultLineGap
	// Honour a configured height when it parses and is in range
	if raw := os.Getenv(lineGapEnv); raw != "" {
		parsed, err := strconv.Atoi(raw)
		// A malformed or out-of-range value keeps the default
		if err == nil && parsed >= 0 && parsed <= maxLineGap {
			gap = parsed
		}
	}
	return strings.Repeat("\n", gap)
}

// Colour depth selection.
const (
	// colorsEnv forces the colour depth: "truecolor" or "256". Unset, the
	// depth follows COLORTERM.
	colorsEnv string = "STATUSLINE_COLORS"
	// colorTermEnv is the de facto variable terminals set to announce 24-bit
	// colour.
	colorTermEnv string = "COLORTERM"
)

// trueColor reports whether 24-bit escapes are used, resolved once at startup.
var trueColor bool = trueColorFromEnv()

// trueColorFromEnv resolves the colour depth from the environment.
//
// A terminal that never announces 24-bit colour gets the 256-colour fallback:
// an unsupported 24-bit escape is dropped or misread, and the ink vanishes.
//
// Returns:
//   - bool: true when 24-bit colour is in use
func trueColorFromEnv() bool {
	// An explicit setting wins over detection
	switch strings.ToLower(os.Getenv(colorsEnv)) {
	// Forced 24-bit colour
	case "truecolor", "24bit":
		return true
	// Forced cube colours
	case "256":
		return false
	}
	// Otherwise trust what the terminal announces
	switch strings.ToLower(os.Getenv(colorTermEnv)) {
	// Both spellings are in use
	case "truecolor", "24bit":
		return true
	// Unset or anything else: stay on the cube
	default:
		return false
	}
}

// pickInk returns the ink for the active colour depth.
//
// Params:
//   - trueInk: 24-bit escape
//   - cubeInk: 256-colour fallback escape
//
// Returns:
//   - string: escape to render with
func pickInk(trueInk, cubeInk string) string {
	// Use the tuned ink only where the terminal can show it
	if trueColor {
		return trueInk
	}
	return cubeInk
}
