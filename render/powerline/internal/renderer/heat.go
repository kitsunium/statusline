// Package renderer provides status line rendering.
package renderer

// Context segment colours.
//
// One ground, one ink, whatever the window holds. The percentage next to them
// already says how full it is, and a colour that changes underneath a figure
// that says the same thing is noise on a line read at a glance.
const (
	// BgContext is the grey-blue ground of the context segment.
	BgContext string = "\033[48;5;152m"
	// FgContext is that ground as a foreground, for the powerline separator.
	FgContext string = "\033[38;5;152m"
	// FgContextInk is the steel blue ink on that ground. 4.53:1.
	FgContextInk string = "\033[38;5;24m"
)
