package snapshot_test

import (
	"regexp"
	"testing"
	"unicode/utf8"

	"pgregory.net/rapid"

	"github.com/kitsunium/statusline/snapshot"
)

var keyCharset = regexp.MustCompile(`^[A-Za-z0-9_-]*$`)

// TestPropertyToolKeyCharset (snapshot/property/tool-key-charset).
func TestPropertyToolKeyCharset(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		name := rapid.String().Draw(t, "name")
		key := snapshot.ToolKey(name)
		if !keyCharset.MatchString(key) {
			t.Fatalf("ToolKey(%q) = %q", name, key)
		}
		if utf8.RuneCountInString(key) != utf8.RuneCountInString(name) {
			t.Fatalf("ToolKey(%q) = %q changed the rune count", name, key)
		}
	})
}

func serversGen() *rapid.Generator[snapshot.MCPServers] {
	return rapid.Custom(func(t *rapid.T) snapshot.MCPServers {
		n := rapid.IntRange(0, 6).Draw(t, "n")
		out := make(snapshot.MCPServers, 0, n)
		for i := 0; i < n; i++ {
			out = append(out, snapshot.MCPServer{
				Name:    rapid.StringMatching(`[a-z.]{1,6}`).Draw(t, "name"),
				Enabled: rapid.Bool().Draw(t, "enabled"),
				Plugin:  rapid.SampledFrom([]string{"", "p", "tasks-plugin"}).Draw(t, "plugin"),
			})
		}
		return out
	})
}

// TestPropertyWithBusyKeepsServers (snapshot/property/with-busy-keeps-servers).
func TestPropertyWithBusyKeepsServers(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		servers := serversGen().Draw(t, "servers")
		before := append(snapshot.MCPServers(nil), servers...)
		keys := rapid.SliceOfN(rapid.StringMatching(`(plugin_[a-z-]{1,6}_)?[a-z_]{1,6}`), 0, 4).Draw(t, "keys")
		got := servers.WithBusy(keys)
		if len(got) < len(before) || len(got) > len(before)+len(keys) {
			t.Fatalf("len = %d from %d servers and %d keys", len(got), len(before), len(keys))
		}
		for i := range before {
			if got[i].Name != before[i].Name || got[i].Enabled != before[i].Enabled {
				t.Fatalf("server %d changed: %+v -> %+v", i, before[i], got[i])
			}
			if servers[i] != before[i] {
				t.Fatal("WithBusy mutated its receiver")
			}
		}
		for _, extra := range got[len(before):] {
			if !extra.Busy || !extra.Enabled {
				t.Fatalf("an appended server is not lit and enabled: %+v", extra)
			}
		}
	})
}

// TestPropertyTaskCountsBounded (snapshot/property/task-counts-bounded).
func TestPropertyTaskCountsBounded(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		statuses := rapid.SliceOf(rapid.SampledFrom([]string{snapshot.TaskPending, snapshot.TaskInProgress, snapshot.TaskWaiting, snapshot.TaskCompleted, "odd"})).Draw(t, "statuses")
		var list snapshot.TaskList
		for i, s := range statuses {
			list.Items = append(list.Items, snapshot.TaskItem{ID: string(rune('a' + i%26)), Subject: s, Status: s})
		}
		if list.Done()+list.Active()+list.Waiting() > list.Total() {
			t.Fatalf("counts exceed the total: %+v", list)
		}
		if list.IsActive() != (list.Total() > 0 && list.Done() < list.Total()) {
			t.Fatal("IsActive disagrees with the counts")
		}
		if subject, status := list.Headline(); subject != "" && status == snapshot.TaskCompleted {
			t.Fatal("a completed task headlines the list")
		}
	})
}

// TestPropertyHealthMonotonic (snapshot/property/health-monotonic).
func TestPropertyHealthMonotonic(t *testing.T) {
	all := []string{"operational", "under_maintenance", "degraded_performance", "partial_outage", "major_outage"}
	rapid.Check(t, func(t *rapid.T) {
		states := rapid.SliceOfN(rapid.SampledFrom(all), 1, 8).Draw(t, "states")
		worse := rapid.SampledFrom(all[2:]).Draw(t, "worse")
		before := snapshot.ClassifyHealth(states)
		after := snapshot.ClassifyHealth(append(append([]string(nil), states...), worse))
		if after < before {
			t.Fatalf("adding %q improved %v to %v", worse, before, after)
		}
	})
}
