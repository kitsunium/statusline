package snapshot_test

import (
	"testing"

	"github.com/kitsunium/statusline/snapshot"
)

func TestGitStatus_IsInRepo(t *testing.T) {
	tests := []struct {
		name   string
		status snapshot.GitStatus
		want   bool
	}{
		{name: "empty branch", status: snapshot.GitStatus{Branch: ""}, want: false},
		{name: "with branch", status: snapshot.GitStatus{Branch: "main"}, want: true},
		{name: "feature branch", status: snapshot.GitStatus{Branch: "feature/test"}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsInRepo(); got != tt.want {
				t.Errorf("IsInRepo() = %v, want %v", got, tt.want)
			}
		})
	}
}
