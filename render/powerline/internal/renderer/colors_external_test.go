package renderer_test

import (
	"testing"

	"github.com/kitsunium/statusline/render/powerline/internal/renderer"
)

func TestGetModelColors(t *testing.T) {
	tests := []struct {
		name      string
		modelName string
		wantBg    string
		wantFg    string
		wantText  string
	}{
		{name: "haiku model", modelName: "Haiku 3.5", wantBg: renderer.BgHaiku, wantFg: renderer.FgHaiku, wantText: renderer.FgHaikuDark},
		{name: "sonnet model", modelName: "Sonnet 3.5", wantBg: renderer.BgSonnet, wantFg: renderer.FgSonnet, wantText: renderer.FgSonnetDark},
		{name: "opus model", modelName: "Opus 4.5", wantBg: renderer.BgOpus, wantFg: renderer.FgOpus, wantText: renderer.FgOpusDark},
		{name: "unknown model", modelName: "Unknown", wantBg: renderer.BgModelUnknown, wantFg: renderer.FgModelUnknown, wantText: renderer.FgModelUnknownDark},
		{name: "fable model", modelName: "Fable 5.1", wantBg: renderer.BgFable, wantFg: renderer.FgFable, wantText: renderer.FgFableDark},
		{name: "case insensitive", modelName: "OPUS", wantBg: renderer.BgOpus, wantFg: renderer.FgOpus, wantText: renderer.FgOpusDark},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bg, fg, text := renderer.GetModelColors(tt.modelName)
			if bg != tt.wantBg || fg != tt.wantFg || text != tt.wantText {
				t.Errorf("GetModelColors(%q) = (%q, %q, %q), want (%q, %q, %q)", tt.modelName, bg, fg, text, tt.wantBg, tt.wantFg, tt.wantText)
			}
		})
	}
}

func TestGetModelColors_EveryFamilyDiffersFromTheOSSegment(t *testing.T) {
	tests := []struct {
		name  string
		model string
	}{
		{name: "opus", model: "Opus 5 (1M context)"},
		{name: "sonnet", model: "Sonnet 5"},
		{name: "haiku", model: "Haiku 4.5"},
		{name: "fable", model: "Fable 5.1"},
		{name: "unknown", model: "Nimbus 2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bg, _, _ := renderer.GetModelColors(tt.model)
			// The OS segment is drawn on white; a model sharing that background
			// merges with it, separator included, into one unreadable block
			if bg == renderer.BgWhite {
				t.Errorf("GetModelColors(%q) background is the OS segment's white", tt.model)
			}
		})
	}
}

func TestGetModelColors_FamiliesAreDistinct(t *testing.T) {
	seen := make(map[string]string, 4)
	for _, name := range []string{"Opus 5", "Sonnet 5", "Haiku 4.5", "Fable 5.1"} {
		bg, _, _ := renderer.GetModelColors(name)
		// Two families sharing a background cannot be told apart at a glance
		if other, clash := seen[bg]; clash {
			t.Errorf("GetModelColors(%q) shares its background with %q", name, other)
		}
		seen[bg] = name
	}
}

func TestGetModelColors_UnknownFallsBackToNeutral(t *testing.T) {
	bg, _, _ := renderer.GetModelColors("Nimbus 2")
	// An unrecognised model has no hue of its own and keeps the neutral default
	if bg != renderer.BgModelUnknown {
		t.Errorf("GetModelColors() for an unknown model = %q, want the neutral default", bg)
	}
	// That default must still differ from the OS segment it sits against
	if bg == renderer.BgWhite {
		t.Error("the unknown-model default is the OS segment's white")
	}
}

func TestNoModelSharesTheOSSegmentWhite(t *testing.T) {
	// The OS segment is white. A model sharing that background merges with the
	// segment before it, separator included, into one unreadable block.
	for _, name := range []string{"Opus 5", "Sonnet 5", "Haiku 4.5", "Fable 5.1", "Nimbus 2"} {
		bg, _, _ := renderer.GetModelColors(name)
		if bg == renderer.BgWhite {
			t.Errorf("GetModelColors(%q) is the OS segment's white", name)
		}
	}
}
