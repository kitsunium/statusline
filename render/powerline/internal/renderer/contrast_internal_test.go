package renderer

import (
	"fmt"
	"math"
	"strings"
	"testing"
)

// ansiToRGB resolves an xterm-256 or 24-bit colour escape to its RGB triple,
// foreground (38) or background (48) alike.
func ansiToRGB(t *testing.T, esc string) (r, g, b float64) {
	t.Helper()
	// A bold prefix changes the weight, not the colour
	esc = strings.Replace(esc, "\033[1;", "\033[", 1)
	var kind, n int
	var tr, tg, tb int
	if _, err := fmt.Sscanf(esc, "\033[%d;2;%d;%d;%dm", &kind, &tr, &tg, &tb); err == nil {
		return float64(tr), float64(tg), float64(tb)
	}
	if _, err := fmt.Sscanf(esc, "\033[%d;5;%dm", &kind, &n); err != nil {
		t.Fatalf("cannot parse %q", esc)
	}
	if n >= 232 {
		v := float64(8 + (n-232)*10)
		return v, v, v
	}
	n -= 16
	lv := []float64{0, 95, 135, 175, 215, 255}
	return lv[n/36], lv[(n/6)%6], lv[n%6]
}

// relLum returns the WCAG relative luminance of an RGB triple.
func relLum(r, g, b float64) float64 {
	conv := func(v float64) float64 {
		v /= 255
		if v <= 0.03928 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*conv(r) + 0.7152*conv(g) + 0.0722*conv(b)
}

// contrast returns the WCAG ratio between two xterm-256 escapes.
func contrast(t *testing.T, fg, bg string) float64 {
	t.Helper()
	fr, fg2, fb := ansiToRGB(t, fg)
	br, bg2, bb := ansiToRGB(t, bg)
	lf, lb := relLum(fr, fg2, fb), relLum(br, bg2, bb)
	hi, lo := math.Max(lf, lb), math.Min(lf, lb)
	return (hi + 0.05) / (lo + 0.05)
}

func TestEverySegmentInkIsReadableOnItsOwnGround(t *testing.T) {
	// A status line is glanced at, never studied. Pastel ink on a pastel ground
	// was the project's original palette and measured as low as 2.08:1 — below
	// the 3:1 floor for large text, let alone the 4.5:1 one for body text.
	const minRatio = 4.5

	tests := []struct {
		name string
		bg   string
		fg   string
	}{
		{name: "opus", bg: BgOpus, fg: inkOpusTrue},
		{name: "opus 256", bg: BgOpus, fg: inkOpus256},
		{name: "sonnet", bg: BgSonnet, fg: inkSonnetTrue},
		{name: "sonnet 256", bg: BgSonnet, fg: inkSonnet256},
		{name: "haiku", bg: BgHaiku, fg: inkHaikuTrue},
		{name: "haiku 256", bg: BgHaiku, fg: inkHaiku256},
		{name: "fable", bg: BgFable, fg: inkFableTrue},
		{name: "fable 256", bg: BgFable, fg: inkFable256},
		{name: "unknown model", bg: BgModelUnknown, fg: FgModelUnknownDark},
		{name: "context", bg: BgContext, fg: FgContextInk},
		{name: "path", bg: BgBlue, fg: FgBlueDark},
		{name: "git", bg: BgGit, fg: FgGitInk},
		{name: "lines added", bg: BgGreen, fg: FgGreenText},
		{name: "lines removed", bg: BgRed, fg: FgRedText},
		{name: "mcp pill", bg: BgMCPEnabled, fg: FgMCPEnabledText},
		{name: "mcp pill, disabled count", bg: BgMCPEnabled, fg: FgMCPMuted},
		{name: "mcp pill during a call", bg: BgMCPLabel, fg: FgWhite},
		{name: "mcp pill during a call, disabled count", bg: BgMCPLabel, fg: FgMCPEnabled},
		{name: "mcp glyph in the os segment", bg: BgWhite, fg: FgMCPOnWhite},
		{name: "mcp count in the os segment", bg: BgWhite, fg: FgBlack},
		{name: "mcp disabled count in the os segment", bg: BgWhite, fg: FgMCPMutedOnWhite},
		{name: "epic pill", bg: BgEpic, fg: FgEpicInk},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := contrast(t, tt.fg, tt.bg); got < minRatio {
				t.Errorf("contrast %.2f:1 is below the %.1f:1 floor", got, minRatio)
			}
		})
	}
}

func TestAdjacentSegmentsDifferInGround(t *testing.T) {
	// Two neighbours sharing a background merge into one block, separator
	// included: the line then reads as fewer segments than it has.
	tests := []struct {
		name  string
		left  string
		right string
	}{
		{name: "os and context are separated by the model", left: BgContext, right: BgBlue},
		{name: "context and path", left: BgContext, right: BgBlue},
		{name: "context and git, the path given up", left: BgContext, right: BgGit},
		{name: "path and git", left: BgBlue, right: BgGit},
		{name: "git and lines added", left: BgGit, right: BgGreen},
		{name: "lines added and removed", left: BgGreen, right: BgRed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.left == tt.right {
				t.Errorf("both segments are drawn on %q", tt.left)
			}
		})
	}
}

func TestHealthGlyphIsReadableOnTheOSGround(t *testing.T) {
	// The health glyph is a graphical object, held to the 3:1 floor.
	const minRatio = 3.0
	for name, fg := range map[string]string{"ok": FgHealthOK, "degraded": FgHealthDegraded, "down": FgHealthDown} {
		t.Run(name, func(t *testing.T) {
			if got := contrast(t, fg, BgWhite); got < minRatio {
				t.Errorf("contrast %.2f:1 is below the %.1f:1 floor", got, minRatio)
			}
		})
	}
}

func TestEpicCellsAreVisibleOnTheMauve(t *testing.T) {
	// The cells are graphical objects: done and under way, in both frames and
	// both colour depths, hold the 3:1 floor. The track of the cells not under
	// way is deliberately pale, yet must not melt into the ground.
	tests := []struct {
		name string
		fg   string
		min  float64
	}{
		{name: "done", fg: FgEpicInk, min: 3.0},
		{name: "under way", fg: epicActiveTrue, min: 3.0},
		{name: "under way 256", fg: epicActive256, min: 3.0},
		{name: "pulse", fg: epicPulseTrue, min: 3.0},
		{name: "pulse 256", fg: epicPulse256, min: 3.0},
		{name: "track", fg: epicTrackTrue, min: 1.4},
		{name: "track 256", fg: epicTrack256, min: 1.4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := contrast(t, tt.fg, BgEpic); got < tt.min {
				t.Errorf("contrast %.2f:1 is below the %.1f:1 floor", got, tt.min)
			}
		})
	}
}

func TestEpicPulseFramesDiffer(t *testing.T) {
	// A pulse whose two frames are the same escape does not pulse
	if epicActiveTrue == epicPulseTrue || epicActive256 == epicPulse256 {
		t.Error("the plain and bright frames of the cell under way are identical")
	}
}
