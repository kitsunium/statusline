package show

import (
	"context"

	"github.com/kitsunium/statusline/ipc"
	"github.com/kitsunium/statusline/quota"
	"github.com/kitsunium/statusline/render/powerline"
	"github.com/kitsunium/statusline/render/session"
)

// execute keys the session the way the daemon keeps it, then draws: the
// directory shown is where the session works (from its transcript), falling
// back on the one the host reported.
func (s *Shower) execute(ctx context.Context, in ShowInput) (ShowOutput, error) {
	getenv := in.Getenv
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	payload := session.Parse(in.Stdin)
	key := ipc.Key{
		SessionID:      payload.SessionID,
		TranscriptPath: payload.Transcript,
		SessionDir:     payload.WorkingDir(),
		TaskListID:     in.TaskListID,
	}
	snap, origin := s.deps.Snapshots.Snapshot(ctx, key)
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
		Icons:       powerline.IconsFromEnv(getenv),
		Width:       powerline.ParseWidth(in.Columns),
		Dir:         dir,
		Git:         snap.Git,
		Changes:     snap.Changes,
		System:      snap.System,
		MCP:         snap.MCP,
		Health:      snap.Health,
		Tasks:       snap.Tasks,
		Working:     snap.Working,
		Update:      snap.Update,
		Now:         s.deps.Clock.Now(),
	}
	return ShowOutput{Line: powerline.Render(frame), Origin: origin}, nil
}
