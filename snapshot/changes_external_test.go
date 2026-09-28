package snapshot_test

import (
	"testing"

	"github.com/kitsunium/statusline/snapshot"
)

func TestCodeChanges_HasChanges(t *testing.T) {
	tests := []struct {
		name    string
		changes snapshot.Changes
		want    bool
	}{
		{name: "no changes", changes: snapshot.Changes{Added: 0, Removed: 0}, want: false},
		{name: "only added", changes: snapshot.Changes{Added: 5, Removed: 0}, want: true},
		{name: "only removed", changes: snapshot.Changes{Added: 0, Removed: 3}, want: true},
		{name: "both", changes: snapshot.Changes{Added: 10, Removed: 5}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.changes.HasChanges(); got != tt.want {
				t.Errorf("HasChanges() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCodeChanges_HasAdded(t *testing.T) {
	tests := []struct {
		name    string
		changes snapshot.Changes
		want    bool
	}{
		{name: "zero added", changes: snapshot.Changes{Added: 0}, want: false},
		{name: "positive added", changes: snapshot.Changes{Added: 10}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.changes.HasAdded(); got != tt.want {
				t.Errorf("HasAdded() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCodeChanges_HasRemoved(t *testing.T) {
	tests := []struct {
		name    string
		changes snapshot.Changes
		want    bool
	}{
		{name: "zero removed", changes: snapshot.Changes{Removed: 0}, want: false},
		{name: "positive removed", changes: snapshot.Changes{Removed: 5}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.changes.HasRemoved(); got != tt.want {
				t.Errorf("HasRemoved() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewCodeChanges(t *testing.T) {
	tests := []struct {
		name    string
		added   int
		removed int
	}{
		{name: "zero values", added: 0, removed: 0},
		{name: "positive values", added: 10, removed: 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := snapshot.Changes{Added: tt.added, Removed: tt.removed}
			if c.Added != tt.added || c.Removed != tt.removed {
				t.Errorf("NewCodeChanges() = {%d, %d}, want {%d, %d}", c.Added, c.Removed, tt.added, tt.removed)
			}
		})
	}
}
