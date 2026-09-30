package quota_test

import (
	"testing"

	"github.com/kitsunium/statusline/quota"
)

func TestNewProgress(t *testing.T) {
	tests := []struct {
		name        string
		totalTokens int
		contextSize int
		wantPercent int
	}{
		{name: "zero context", totalTokens: 100, contextSize: 0, wantPercent: 0},
		{name: "50 percent", totalTokens: 100, contextSize: 200, wantPercent: 50},
		{name: "100 percent", totalTokens: 200, contextSize: 200, wantPercent: 100},
		{name: "over 100 capped", totalTokens: 300, contextSize: 200, wantPercent: 100},
		{name: "zero tokens", totalTokens: 0, contextSize: 200, wantPercent: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := quota.NewProgress(tt.totalTokens, tt.contextSize)
			if p.Percent != tt.wantPercent {
				t.Errorf("NewProgress() Percent = %d, want %d", p.Percent, tt.wantPercent)
			}
		})
	}
}

func TestProgress_Level(t *testing.T) {
	tests := []struct {
		name    string
		percent int
		want    quota.Level
	}{
		{name: "low 0", percent: 0, want: quota.LevelLow},
		{name: "low 49", percent: 49, want: quota.LevelLow},
		{name: "medium 50", percent: 50, want: quota.LevelMedium},
		{name: "medium 74", percent: 74, want: quota.LevelMedium},
		{name: "high 75", percent: 75, want: quota.LevelHigh},
		{name: "high 89", percent: 89, want: quota.LevelHigh},
		{name: "critical 90", percent: 90, want: quota.LevelCritical},
		{name: "critical 100", percent: 100, want: quota.LevelCritical},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := quota.Progress{Percent: tt.percent}
			if got := p.Level(); got != tt.want {
				t.Errorf("Level() = %v, want %v", got, tt.want)
			}
		})
	}
}
