package session_test

import (
	"testing"

	"github.com/kitsunium/statusline/render/session"
)

func TestInput_ModelInfo(t *testing.T) {
	tests := []struct {
		name        string
		input       session.Payload
		wantName    string
		wantVersion string
	}{
		{
			name:        "empty display name uses default",
			input:       session.Payload{},
			wantName:    "Claude",
			wantVersion: "",
		},
		{
			name:        "name with version",
			input:       session.Payload{Model: session.PayloadModel{DisplayName: "Opus 4.5"}},
			wantName:    "Opus",
			wantVersion: "4.5",
		},
		{
			name:        "name without version",
			input:       session.Payload{Model: session.PayloadModel{DisplayName: "Sonnet"}},
			wantName:    "Sonnet",
			wantVersion: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := tt.input.ModelInfo()
			if info.Name != tt.wantName || info.Version != tt.wantVersion {
				t.Errorf("ModelInfo() = {%q, %q}, want {%q, %q}", info.Name, info.Version, tt.wantName, tt.wantVersion)
			}
		})
	}
}

func TestInput_WorkingDir(t *testing.T) {
	tests := []struct {
		name  string
		input session.Payload
		want  string
	}{
		{name: "empty uses default", input: session.Payload{}, want: "~"},
		{name: "custom dir", input: session.Payload{Workspace: session.PayloadWorkspace{CurrentDir: "/workspace"}}, want: "/workspace"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.input.WorkingDir(); got != tt.want {
				t.Errorf("WorkingDir() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInput_ContextWindowSize(t *testing.T) {
	tests := []struct {
		name  string
		input session.Payload
		want  int
	}{
		{name: "zero uses default", input: session.Payload{}, want: 200000},
		{name: "custom size", input: session.Payload{ContextWindow: session.PayloadContext{ContextWindowSize: 100000}}, want: 100000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.input.ContextWindowSize(); got != tt.want {
				t.Errorf("ContextWindowSize() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestInput_TotalTokens(t *testing.T) {
	tests := []struct {
		name  string
		input session.Payload
		want  int
	}{
		{name: "zero tokens", input: session.Payload{}, want: 0},
		{name: "sum of tokens", input: session.Payload{ContextWindow: session.PayloadContext{TotalInputTokens: 100, TotalOutputTokens: 50}}, want: 150},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.input.TotalTokens(); got != tt.want {
				t.Errorf("TotalTokens() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestInput_Progress(t *testing.T) {
	pct := func(v float64) *float64 { return &v }
	tests := []struct {
		name        string
		input       session.Payload
		wantPercent int
	}{
		{name: "zero tokens fallback", input: session.Payload{ContextWindow: session.PayloadContext{ContextWindowSize: 200000}}, wantPercent: 0},
		{name: "token fallback 50 percent", input: session.Payload{ContextWindow: session.PayloadContext{TotalInputTokens: 100000, ContextWindowSize: 200000}}, wantPercent: 50},
		{name: "used_percentage preferred", input: session.Payload{ContextWindow: session.PayloadContext{UsedPercentage: pct(39), TotalInputTokens: 500000, ContextWindowSize: 200000}}, wantPercent: 39},
		{name: "used_percentage zero", input: session.Payload{ContextWindow: session.PayloadContext{UsedPercentage: pct(0)}}, wantPercent: 0},
		{name: "used_percentage capped at 100", input: session.Payload{ContextWindow: session.PayloadContext{UsedPercentage: pct(150)}}, wantPercent: 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.input.Progress()
			if p.Percent != tt.wantPercent {
				t.Errorf("Progress() = %d%%, want %d%%", p.Percent, tt.wantPercent)
			}
		})
	}
}
