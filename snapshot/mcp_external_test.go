package snapshot_test

import (
	"testing"

	"github.com/kitsunium/statusline/snapshot"
)

func TestMCPServersWithBusy(t *testing.T) {
	inv := snapshot.MCPServers{
		{Name: "github", Enabled: true},
		{Name: "tasks", Enabled: true, Plugin: "kodflow-hooks"},
		{Name: "claude.ai Docs", Enabled: true},
		{Name: "off", Enabled: false},
	}
	tests := []struct {
		name string
		keys []string
		busy []string
		add  string
	}{
		{name: "none", keys: nil},
		{name: "plain", keys: []string{"github"}, busy: []string{"github"}},
		{name: "plugin key", keys: []string{"plugin_kodflow-hooks_tasks"}, busy: []string{"tasks"}},
		{name: "plugin key of an unknown plugin", keys: []string{"plugin_other_tasks"}, busy: []string{"tasks"}},
		{name: "normalised name", keys: []string{"claude_ai_Docs"}, busy: []string{"claude.ai Docs"}},
		{name: "disabled still lights", keys: []string{"off"}, busy: []string{"off"}},
		{name: "unknown server is added", keys: []string{"plugin_x_new_srv"}, busy: []string{"new_srv"}, add: "new_srv"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inv.WithBusy(tt.keys)
			want := map[string]bool{}
			for _, n := range tt.busy {
				want[n] = true
			}
			for _, s := range got {
				if s.Busy != want[s.Name] {
					t.Errorf("%s: Busy = %v, want %v", s.Name, s.Busy, want[s.Name])
				}
			}
			if tt.add != "" {
				last := got[len(got)-1]
				if len(got) != len(inv)+1 || last.Name != tt.add || !last.Enabled {
					t.Errorf("unknown server not appended enabled: %+v", got)
				}
			} else if len(got) != len(inv) {
				t.Errorf("len = %d, want %d", len(got), len(inv))
			}
		})
	}
	if inv[0].Busy {
		t.Error("WithBusy modified its receiver")
	}
}

func TestToolKey(t *testing.T) {
	if got := snapshot.ToolKey("claude.ai Claude Docs"); got != "claude_ai_Claude_Docs" {
		t.Errorf("ToolKey = %q", got)
	}
	if got := snapshot.ToolKey("kodflow-hooks_1"); got != "kodflow-hooks_1" {
		t.Errorf("ToolKey = %q", got)
	}
}

func TestMCPServersWithSource(t *testing.T) {
	servers := snapshot.MCPServers{{Name: "a"}, {Name: "b", Source: snapshot.MCPSourceUser}}
	got := servers.WithSource(snapshot.MCPSourcePlugin)
	for _, s := range got {
		if s.Source != snapshot.MCPSourcePlugin {
			t.Errorf("%s: Source = %q, want plugin", s.Name, s.Source)
		}
	}
	if len(snapshot.MCPServers(nil).WithSource(snapshot.MCPSourceCLI)) != 0 {
		t.Error("an empty list stays empty")
	}
	// An unknown server added by WithBusy carries no scope
	busy := snapshot.MCPServers{{Name: "a", Enabled: true, Source: snapshot.MCPSourceCLI}}.WithBusy([]string{"ghost"})
	if busy[1].Source != snapshot.MCPSourceUnknown {
		t.Errorf("ghost: Source = %q, want unknown", busy[1].Source)
	}
}
