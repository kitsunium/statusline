package gather_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/kitsunium/statusline/collect/gather"
	"github.com/kitsunium/statusline/collect/refresh"
	"github.com/kitsunium/statusline/collect/state"
	"github.com/kitsunium/statusline/ipc"
	"github.com/kitsunium/statusline/quota"
	"github.com/kitsunium/statusline/snapshot"
)

// sources answers every collect port and counts the calls.
type sources struct {
	mu     sync.Mutex
	now    time.Time
	calls  map[string]int
	gitDir string
	cached []ipc.Key
}

func (s *sources) hit(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls[name]++
}

func (s *sources) Now() time.Time { return s.now }

func (s *sources) Dir(transcript, fallback string) string {
	s.hit("workdir")
	if transcript == "" {
		return fallback
	}
	return "/work/elsewhere"
}

func (s *sources) Lookup(string) (state.HostSession, bool) {
	s.hit("sessions")
	return state.HostSession{PID: 42, Working: true}, true
}

func (s *sources) Status(dir string) snapshot.GitStatus {
	s.hit("git")
	s.mu.Lock()
	s.gitDir = dir
	s.mu.Unlock()
	return snapshot.GitStatus{Branch: "main"}
}

func (s *sources) DiffStats(string) snapshot.Changes {
	s.hit("diff")
	return snapshot.Changes{Added: 1}
}

func (s *sources) Servers(projectDir string, pid int) snapshot.MCPServers {
	s.hit("mcp")
	if pid != 42 {
		return nil
	}
	return snapshot.MCPServers{{Name: "github", Enabled: true}}
}

func (s *sources) Busy(string, string, time.Time) []string {
	s.hit("calls")
	return []string{"github"}
}

func (s *sources) Board(string, string, time.Time) snapshot.TaskBoard {
	s.hit("tasks")
	return snapshot.TaskBoard{Unattributed: 1}
}

func (s *sources) Info() snapshot.System {
	s.hit("system")
	return snapshot.System{OS: snapshot.OSDarwin}
}

func (s *sources) SaveSnapshot(key ipc.Key, _ snapshot.Snapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cached = append(s.cached, key)
	return nil
}

// network is a LatestNetwork with fixed figures.
type network struct{}

func (network) Execute(context.Context, refresh.LatestInput) (refresh.LatestOutput, error) {
	set := quota.Set{Scoped: []quota.Limit{quota.NewLimit(quota.KindScoped, "opus", 5, time.Unix(1, 0), quota.WeeklyWindow, quota.SourceAPI)}}
	return refresh.LatestOutput{Usage: set, HasUsage: true, Health: snapshot.HealthDegraded}, nil
}

func newCollector(s *sources) *gather.Collector {
	return gather.New(gather.Deps{
		Clock: s, WorkDir: s, Sessions: s, Repository: s, MCPConfig: s, MCPCalls: s,
		Tasks: s, System: s, Cache: s, Network: network{}, Registry: state.NewRegistry(),
	})
}

func TestCollectThenFreshHit(t *testing.T) {
	s := &sources{now: time.Unix(1_800_000_000, 0), calls: map[string]int{}}
	c := newCollector(s)
	key := ipc.Key{SessionID: "s", TranscriptPath: "/t.jsonl", SessionDir: "/work/session"}
	out, err := c.Execute(context.Background(), gather.CollectInput{Key: key})
	if err != nil || !out.Collected {
		t.Fatalf("Execute() = %+v, %v", out, err)
	}
	snap := out.Snapshot
	if snap.WorkDir != "/work/elsewhere" || s.gitDir != "/work/elsewhere" {
		t.Errorf("git must run where the session works: workdir %q, git in %q", snap.WorkDir, s.gitDir)
	}
	if !snap.Working || len(snap.MCP) != 1 || !snap.MCP[0].Busy || snap.System.OS != snapshot.OSDarwin {
		t.Errorf("snapshot = %+v", snap)
	}
	if snap.Health != snapshot.HealthDegraded || len(snap.API.Scoped) != 1 {
		t.Errorf("network figures missing: %+v", snap)
	}
	if len(s.cached) != 1 || s.cached[0] != key {
		t.Errorf("cached = %v, want the key once", s.cached)
	}

	// fresh-hit: within Freshness nothing but the clock is asked
	before := len(s.calls)
	total := func() int {
		n := 0
		for _, v := range s.calls {
			n += v
		}
		return n
	}
	calls := total()
	s.now = s.now.Add(state.Freshness / 2)
	out, _ = c.Execute(context.Background(), gather.CollectInput{Key: key})
	if out.Collected || total() != calls || len(s.calls) != before {
		t.Errorf("a fresh snapshot was collected again: %v", s.calls)
	}
	s.now = s.now.Add(state.Freshness)
	if out, _ = c.Execute(context.Background(), gather.CollectInput{Key: key}); !out.Collected {
		t.Error("a stale snapshot was served")
	}
}

func TestRelativeWorkDirKeepsGitInPlace(t *testing.T) {
	s := &sources{now: time.Unix(1_800_000_000, 0), calls: map[string]int{}}
	_, _ = newCollector(s).Execute(context.Background(), gather.CollectInput{Key: ipc.Key{SessionDir: "~"}})
	if s.gitDir != "" {
		t.Errorf("git ran in %q for a relative directory, want the daemon's own", s.gitDir)
	}
}
