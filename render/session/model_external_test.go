package session_test

import (
	"testing"

	"github.com/kitsunium/statusline/render/session"
)

func TestModelInfo_FullName(t *testing.T) {
	tests := []struct {
		name string
		info session.ModelInfo
		want string
	}{
		{name: "name only", info: session.ModelInfo{Name: "Claude"}, want: "Claude"},
		{name: "name with version", info: session.ModelInfo{Name: "Opus", Version: "4.5"}, want: "Opus 4.5"},
		{name: "empty name", info: session.ModelInfo{Name: "", Version: "1.0"}, want: " 1.0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.info.FullName(); got != tt.want {
				t.Errorf("FullName() = %q, want %q", got, tt.want)
			}
		})
	}
}
