// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package show

import (
	"context"

	"github.com/kitsunium/statusline/ipc"
	"github.com/kitsunium/statusline/quota"
	"github.com/kitsunium/statusline/render/powerline"
	"github.com/kitsunium/statusline/render/session"
	"github.com/kitsunium/statusline/snapshot"
)

// execute keys the session the way the daemon keeps it, then draws: the
// directory shown is where the session works (from its transcript), falling
// back on the one the host reported. It never fails: a source that errs
// leaves what stdin alone establishes.
func (h *ShowStatusLineHandler) execute(ctx context.Context, in ShowStatusLineInput) (ShowStatusLineOutput, error) {
	payload := session.Parse(in.Stdin)
	key := ipc.Key{
		SessionID:      payload.SessionID,
		TranscriptPath: payload.Transcript,
		SessionDir:     payload.WorkingDir(),
		TaskListID:     in.Env["CLAUDE_CODE_TASK_LIST_ID"],
	}
	snap, origin, err := h.snapshotSource.Snapshot(ctx, key)
	if err != nil {
		snap, origin = snapshot.Snapshot{}, OriginNone
	}
	now, _ := h.clock.Now(ctx)
	dir := snap.WorkDir
	if dir == "" {
		dir = payload.WorkingDir()
	}
	frame := powerline.Frame{
		Model:       payload.ModelInfo(),
		Effort:      payload.Effort.Level,
		FastMode:    payload.FastMode,
		SessionName: payload.SessionName,
		Cost:        payload.Cost.TotalCostUSD,
		Limits:      quota.Resolve(payload.StdinLimits(), snap.API),
		Icons:       powerline.IconsFromEnv(in.Env),
		Width:       powerline.ParseWidth(in.Env["COLUMNS"]),
		Dir:         dir,
		Git:         snap.Git,
		Changes:     snap.Changes,
		System:      snap.System,
		MCP:         snap.MCP,
		Health:      snap.Health,
		Tasks:       snap.Tasks,
		Working:     snap.Working,
		Update:      snap.Update,
		Now:         now,
	}
	return ShowStatusLineOutput{Line: powerline.Render(frame), Origin: origin}, nil
}
