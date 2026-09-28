package gather

import (
	"context"
	"path/filepath"
	"sync"

	"github.com/kitsunium/statusline/collect/refresh"
	"github.com/kitsunium/statusline/snapshot"
)

// execute serves the registry's snapshot while it is fresh; otherwise it
// collects, keeps and caches a new one. Every request counts against the
// key's eviction.
func (c *Collector) execute(ctx context.Context, in CollectInput) (CollectOutput, error) {
	now := c.deps.Clock.Now()
	c.deps.Registry.Touch(in.Key, now)
	if snap, ok := c.deps.Registry.Get(in.Key, now); ok {
		return CollectOutput{Snapshot: snap}, nil
	}
	snap := c.collect(ctx, in)
	snap.CollectedAt = now
	c.deps.Registry.Put(in.Key, snap, now)
	// A cache write failure only costs the next cold render
	_ = c.deps.Cache.SaveSnapshot(in.Key, snap)
	return CollectOutput{Snapshot: snap, Collected: true}, nil
}

// collect gathers every source concurrently: each is a small read or one
// git process, and in sequence their latencies would add up.
//
// MCP servers are configured for the session's own directory, while git
// follows wherever the session is actually working.
func (c *Collector) collect(ctx context.Context, in CollectInput) snapshot.Snapshot {
	key := in.Key
	now := c.deps.Clock.Now()
	workDir := c.deps.WorkDir.Dir(key.TranscriptPath, key.SessionDir)
	gitDir := workDir
	// Without a real directory git keeps the daemon's working directory
	if !filepath.IsAbs(gitDir) {
		gitDir = ""
	}
	host, _ := c.deps.Sessions.Lookup(key.SessionID)

	var (
		wg   sync.WaitGroup
		snap = snapshot.Snapshot{WorkDir: workDir, Working: host.Working}
	)
	gather := func(fn func()) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fn()
		}()
	}
	gather(func() { snap.Git = c.deps.Repository.Status(gitDir) })
	gather(func() { snap.Changes = c.deps.Repository.DiffStats(gitDir) })
	gather(func() { snap.System = c.deps.System.Info() })
	gather(func() {
		busy := c.deps.MCPCalls.Busy(key.TranscriptPath, key.SessionID, now)
		snap.MCP = c.deps.MCPConfig.Servers(key.SessionDir, host.PID).WithBusy(busy)
	})
	gather(func() { snap.Tasks = c.deps.Tasks.Board(key.SessionID, key.TaskListID, now) })
	wg.Wait()

	if latest, err := c.deps.Network.Execute(ctx, refresh.LatestInput{}); err == nil {
		if latest.HasUsage {
			snap.API = latest.Usage
		}
		snap.Health = latest.Health
	}
	return snap
}
