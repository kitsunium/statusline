package renderer

import (
	"testing"

	"github.com/kitsunium/statusline/render/powerline/internal/model"
)

func TestRenderHeavyBar(t *testing.T) {
	tests := []struct {
		name    string
		percent int
		wantLen int
	}{
		{name: "0 percent", percent: 0, wantLen: progressBarWidth},
		{name: "50 percent", percent: 50, wantLen: progressBarWidth},
		{name: "100 percent", percent: 100, wantLen: progressBarWidth},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			progress := model.Progress{Percent: tt.percent}
			result := renderHeavyBar(progress, noCursor)
			if len([]rune(result)) != tt.wantLen {
				t.Errorf("renderHeavyBar() len = %d, want %d", len([]rune(result)), tt.wantLen)
			}
		})
	}
}

func TestRenderGranularBar(t *testing.T) {
	tests := []struct {
		name    string
		percent int
		style   ProgressBarStyle
		wantLen int
	}{
		{name: "braille 50%", percent: 50, style: StyleBraille, wantLen: progressBarWidth},
		{name: "block 75%", percent: 75, style: StyleBlock, wantLen: progressBarWidth},
		{name: "braille 0%", percent: 0, style: StyleBraille, wantLen: progressBarWidth},
		{name: "block 100%", percent: 100, style: StyleBlock, wantLen: progressBarWidth},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			progress := model.Progress{Percent: tt.percent}
			result := renderGranularBar(progress, tt.style)
			if len([]rune(result)) != tt.wantLen {
				t.Errorf("renderGranularBar() len = %d, want %d", len([]rune(result)), tt.wantLen)
			}
		})
	}
}

func TestRenderHeavyBarWithCursor(t *testing.T) {
	tests := []struct {
		name      string
		percent   int
		cursorIdx int
	}{
		{name: "cursor at start", percent: 50, cursorIdx: 0},
		{name: "cursor in middle", percent: 50, cursorIdx: 10},
		{name: "cursor at end", percent: 50, cursorIdx: 19},
		{name: "empty bar with cursor", percent: 0, cursorIdx: 10},
		{name: "full bar with cursor", percent: 100, cursorIdx: 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			progress := model.Progress{Percent: tt.percent}
			result := renderHeavyBarWithCursor(progress, tt.cursorIdx, progressBarWidth, FgWeeklyText, Reset)
			if len(result) == 0 {
				t.Error("renderHeavyBarWithCursor() returned empty string")
			}
		})
	}
}
