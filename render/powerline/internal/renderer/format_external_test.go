package renderer_test

import (
	"testing"
	"time"

	"github.com/kitsunium/statusline/render/powerline/internal/renderer"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{name: "elapsed", d: -time.Minute, want: "now"},
		{name: "zero", d: 0, want: "now"},
		{name: "minutes", d: 45 * time.Minute, want: "45m"},
		{name: "whole hours", d: 3 * time.Hour, want: "3h"},
		{name: "hours and minutes", d: 2*time.Hour + 10*time.Minute, want: "2h10"},
		{name: "hours and single digit minutes", d: 2*time.Hour + 5*time.Minute, want: "2h05"},
		{name: "whole days", d: 3 * 24 * time.Hour, want: "3d"},
		{name: "days and hours", d: 3*24*time.Hour + 5*time.Hour, want: "3d5h"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := renderer.FormatDuration(tt.d); got != tt.want {
				t.Errorf("FormatDuration(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}

func TestFormatTokens(t *testing.T) {
	tests := []struct {
		name   string
		tokens int
		want   string
	}{
		{name: "units", tokens: 812, want: "812"},
		{name: "thousands", tokens: 103550, want: "103k"},
		{name: "exact million", tokens: 1000000, want: "1M"},
		{name: "fractional million", tokens: 1250000, want: "1.2M"},
		{name: "zero", tokens: 0, want: "0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := renderer.FormatTokens(tt.tokens); got != tt.want {
				t.Errorf("FormatTokens(%d) = %q, want %q", tt.tokens, got, tt.want)
			}
		})
	}
}
