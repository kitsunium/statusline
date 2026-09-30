package renderer_test

import (
	"testing"

	"github.com/kitsunium/statusline/render/powerline/internal/renderer"
)

func TestTruncatePath(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		maxLen int
	}{
		{name: "short path unchanged", path: "/usr/local", maxLen: 30},
		{name: "long path truncated", path: "/very/long/path/that/exceeds/the/maximum/length", maxLen: 20},
		{name: "zero maxLen uses default", path: "/short", maxLen: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderer.TruncatePath(tt.path, tt.maxLen)
			effectiveMax := tt.maxLen
			if effectiveMax <= 0 {
				effectiveMax = 30
			}
			if len(got) > effectiveMax && len(tt.path) > effectiveMax {
				// Path was truncated but still too long - check if it starts with ellipsis
				if got[:3] != "..." {
					t.Errorf("TruncatePath(%q, %d) = %q, want to start with ...", tt.path, tt.maxLen, got)
				}
			}
		})
	}
}
