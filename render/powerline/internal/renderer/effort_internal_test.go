package renderer

import (
	"strings"
	"testing"
)

func TestEffortGauge(t *testing.T) {
	const ink, track = "<ink>", "<track>"
	tests := []struct {
		name    string
		level   string
		wantOn  int
		wantOff int
		want    string
	}{
		{name: "low", level: "low", wantOn: 1, wantOff: 4},
		{name: "xhigh", level: "xhigh", wantOn: 4, wantOff: 1},
		{name: "max", level: "max", wantOn: 5, wantOff: 0},
		{name: "unknown level is written out", level: "ultra", want: ink + "· ultra"},
		{name: "no effort draws nothing", level: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EffortGauge(tt.level, ink, track)
			if tt.wantOn == 0 && tt.wantOff == 0 {
				if got != tt.want {
					t.Errorf("EffortGauge(%q) = %q, want %q", tt.level, got, tt.want)
				}
				return
			}
			want := ink + strings.Repeat(glyphs.EffortOn, tt.wantOn) + track + strings.Repeat(glyphs.EffortOff, tt.wantOff) + ink
			if got != want {
				t.Errorf("EffortGauge(%q) = %q, want %q", tt.level, got, want)
			}
		})
	}
}

func TestModelTracksSitBetweenGroundAndInk(t *testing.T) {
	// A track must read as "empty": visible on its ground, yet clearly
	// lighter than the ink of the reached levels.
	tests := []struct {
		name, bg, ink, track string
	}{
		{name: "haiku", bg: BgHaiku, ink: inkHaikuTrue, track: trackHaikuTrue},
		{name: "haiku 256", bg: BgHaiku, ink: inkHaiku256, track: trackHaiku256},
		{name: "sonnet", bg: BgSonnet, ink: inkSonnetTrue, track: trackSonnetTrue},
		{name: "sonnet 256", bg: BgSonnet, ink: inkSonnet256, track: trackSonnet256},
		{name: "opus", bg: BgOpus, ink: inkOpusTrue, track: trackOpusTrue},
		{name: "opus 256", bg: BgOpus, ink: inkOpus256, track: trackOpus256},
		{name: "fable", bg: BgFable, ink: inkFableTrue, track: trackFableTrue},
		{name: "fable 256", bg: BgFable, ink: inkFable256, track: trackFable256},
		{name: "unknown", bg: BgModelUnknown, ink: FgModelUnknownDark, track: trackModelUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			onGround := contrast(t, tt.track, tt.bg)
			if onGround < 1.4 {
				t.Errorf("track at %.2f:1 on its ground is too faint to see", onGround)
			}
			if contrast(t, tt.ink, tt.bg) <= onGround {
				t.Errorf("track is not lighter than the ink")
			}
		})
	}
}
