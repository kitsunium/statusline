package renderer_test

import (
	"strings"
	"testing"

	"github.com/kitsunium/statusline/render/powerline/internal/model"
	"github.com/kitsunium/statusline/render/powerline/internal/renderer"
)

func TestNewPowerline(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "creates renderer"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := renderer.NewPowerline()
			if r == nil {
				t.Error("NewPowerline() returned nil")
			}
		})
	}
}

func TestPowerline_Render(t *testing.T) {
	tests := []struct {
		name string
		data model.StatusLineData
	}{
		{
			name: "with git and changes",
			data: model.StatusLineData{
				Model:    model.ModelInfo{Name: "Opus", Version: "4.5"},
				Progress: model.Progress{Percent: 50},
				Icons:    model.IconConfig{OS: true, Model: true, Path: true, Git: true},
				Git:      model.GitStatus{Branch: "main", Modified: 1},
				System:   model.SystemInfo{OS: model.OSLinux},
				Dir:      "/workspace",
				Changes:  model.CodeChanges{Added: 10, Removed: 5},
			},
		},
		{
			name: "without git",
			data: model.StatusLineData{
				Model:    model.ModelInfo{Name: "Sonnet"},
				Progress: model.Progress{Percent: 25},
				System:   model.SystemInfo{OS: model.OSDarwin},
				Dir:      "/Users/test",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := renderer.NewPowerline()
			result := r.Render(tt.data)
			if result == "" {
				t.Error("Render() returned empty string")
			}
			if !strings.Contains(result, "\n") {
				t.Error("Render() should contain newlines")
			}
		})
	}
}
