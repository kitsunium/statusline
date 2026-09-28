package session_test

import (
	"testing"

	"github.com/kitsunium/statusline/render/session"
)

func TestEffortRank(t *testing.T) {
	tests := []struct {
		level    string
		wantRank int
		wantOK   bool
	}{
		{level: "low", wantRank: 1, wantOK: true},
		{level: "medium", wantRank: 2, wantOK: true},
		{level: "high", wantRank: 3, wantOK: true},
		{level: "xhigh", wantRank: 4, wantOK: true},
		{level: "max", wantRank: 5, wantOK: true},
		{level: "ultra", wantRank: 0, wantOK: false},
		{level: "", wantRank: 0, wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			rank, ok := session.EffortRank(tt.level)
			if rank != tt.wantRank || ok != tt.wantOK {
				t.Errorf("EffortRank(%q) = %d, %v, want %d, %v", tt.level, rank, ok, tt.wantRank, tt.wantOK)
			}
		})
	}
	if got := session.EffortSteps(); got != 5 {
		t.Errorf("EffortSteps() = %d, want 5", got)
	}
}
