// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package collector

import (
	"context"
	"path/filepath"
	"sync"
	"time"

	"github.com/kitsunium/statusline/snapshot"
)

// execute serves the cache's snapshot while it is fresh; otherwise it
// collects, keeps and caches a new one. Every request counts against the
// key's eviction. A source that fails costs only its own part of the line.
func (h *CollectSnapshotHandler) execute(ctx context.Context, in CollectSnapshotInput) (CollectSnapshotOutput, error) {
	now, err := h.clock.Now(ctx)
	if err != nil {
		return CollectSnapshotOutput{}, err
	}
	_ = h.sessionCache.Touch(ctx, in.Key, now)
	if snap, ok, err := h.sessionCache.Get(ctx, in.Key, now); err == nil && ok {
		return CollectSnapshotOutput{Snapshot: snap}, nil
	}
	snap := h.collect(ctx, in, now)
	snap.CollectedAt = now
	_ = h.sessionCache.Put(ctx, in.Key, snap, now)
	// A cache write failure only costs the next cold render
	_ = h.snapshotCache.SaveSnapshot(ctx, in.Key, snap)
	return CollectSnapshotOutput{Snapshot: snap, Collected: true}, nil
}

// collect reads every source concurrently: each is a small read or one git
// process, and in sequence their latencies would add up. MCP servers are
// configured for the session's own directory, while git follows wherever
// the session is actually working.
func (h *CollectSnapshotHandler) collect(ctx context.Context, in CollectSnapshotInput, now time.Time) snapshot.Snapshot {
	key := in.Key
	workDir, err := h.workDir.Dir(ctx, key.TranscriptPath, key.SessionDir)
	if err != nil || workDir == "" {
		workDir = key.SessionDir
	}
	gitDir := workDir
	// Without a real directory git keeps the daemon's working directory
	if !filepath.IsAbs(gitDir) {
		gitDir = ""
	}
	host, _, _ := h.sessionRegistry.Lookup(ctx, key.SessionID)

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
	gather(func() { snap.Git, _ = h.repository.Status(ctx, gitDir) })
	gather(func() { snap.Changes, _ = h.repository.DiffStats(ctx, gitDir) })
	gather(func() { snap.System, _ = h.systemInfo.Info(ctx) })
	gather(func() {
		busy, _ := h.mcpCalls.Busy(ctx, key.TranscriptPath, key.SessionID, now)
		servers, _ := h.mcpConfig.Servers(ctx, key.SessionDir, host.PID)
		snap.MCP = servers.WithBusy(busy)
	})
	gather(func() { snap.Tasks, _ = h.taskStore.Board(ctx, key.SessionID, key.TaskListID, now) })
	wg.Wait()

	if latest, err := h.latestNetwork.Execute(ctx, LatestNetworkInput{}); err == nil {
		if latest.HasUsage {
			snap.API = latest.Usage
		}
		snap.Health = latest.Health
	}
	return snap
}
