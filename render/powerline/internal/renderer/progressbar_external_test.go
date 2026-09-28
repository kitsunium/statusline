package renderer_test

import (
	"testing"

	"github.com/kitsunium/statusline/render/powerline/internal/model"
	"github.com/kitsunium/statusline/render/powerline/internal/renderer"
)

func TestRenderProgressBar(t *testing.T) {
	tests := []struct {
		name    string
		percent int
		style   renderer.ProgressBarStyle
		wantLen int
	}{
		{name: "heavy 0%", percent: 0, style: renderer.StyleHeavy, wantLen: 20},
		{name: "heavy 50%", percent: 50, style: renderer.StyleHeavy, wantLen: 20},
		{name: "heavy 100%", percent: 100, style: renderer.StyleHeavy, wantLen: 20},
		{name: "block 50%", percent: 50, style: renderer.StyleBlock, wantLen: 20},
		{name: "braille 75%", percent: 75, style: renderer.StyleBraille, wantLen: 20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			progress := model.Progress{Percent: tt.percent}
			result := renderer.RenderProgressBar(progress, tt.style)
			if len([]rune(result)) != tt.wantLen {
				t.Errorf("RenderProgressBar() len = %d, want %d", len([]rune(result)), tt.wantLen)
			}
		})
	}
}

func TestRenderProgressBarWithCursor(t *testing.T) {
	tests := []struct {
		name      string
		percent   int
		cursorPos int
	}{
		{name: "cursor at start", percent: 50, cursorPos: 0},
		{name: "cursor in middle", percent: 50, cursorPos: 50},
		{name: "cursor at end", percent: 50, cursorPos: 100},
		{name: "empty bar with cursor", percent: 0, cursorPos: 50},
		{name: "full bar with cursor", percent: 100, cursorPos: 50},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			progress := model.Progress{Percent: tt.percent}
			result := renderer.RenderProgressBarWithCursor(progress, tt.cursorPos, "\033[38;5;166m", "\033[0m")
			if len(result) == 0 {
				t.Error("RenderProgressBarWithCursor() returned empty string")
			}
		})
	}
}
