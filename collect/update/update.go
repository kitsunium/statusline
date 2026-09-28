package update

import (
	"context"
	"errors"
	"fmt"

	"github.com/kitsunium/statusline/ipc"
	"github.com/kitsunium/statusline/snapshot"
)

// execute never elevates and never retries a version that failed its probe.
func (u *Updater) execute(ctx context.Context, in UpdateInput) (UpdateOutput, error) {
	// A development build has nothing to compare, and an opted-out one asks
	// nothing of the network
	if in.Disabled || ipc.CompareVersions(in.CurrentVersion, "v0.0.0") < 0 {
		return UpdateOutput{}, nil
	}
	now := u.deps.Clock.Now()
	st, err := u.deps.Store.LoadUpdate()
	if err != nil {
		return UpdateOutput{}, fmt.Errorf("load update state: %w", err)
	}
	if !st.Due(now) {
		return UpdateOutput{}, nil
	}
	st.CheckedAt = now
	rel, err := u.deps.Releases.Latest(ctx)
	if err != nil || rel.Version == "" || rel.Version == st.BadVersion ||
		ipc.CompareVersions(rel.Version, in.CurrentVersion) <= 0 {
		return UpdateOutput{}, errors.Join(err, u.deps.Store.SaveUpdate(st))
	}
	notice := snapshot.UpdateNotice{Available: true, Version: rel.Version}
	if err := u.deps.Releases.Install(ctx, rel); err != nil {
		st.Failures++
		return UpdateOutput{}, errors.Join(fmt.Errorf("install %s: %w", rel.Version, err), u.deps.Store.SaveUpdate(st))
	}
	if err := u.deps.Releases.Probe(ctx); err != nil {
		rollback := u.deps.Releases.Rollback(ctx)
		st.BadVersion, st.Failures = rel.Version, st.Failures+1
		return UpdateOutput{}, errors.Join(fmt.Errorf("probe %s: %w", rel.Version, err), rollback, u.deps.Store.SaveUpdate(st))
	}
	st.Installed, st.Failures = rel.Version, 0
	return UpdateOutput{Notice: notice, Installed: true}, u.deps.Store.SaveUpdate(st)
}
