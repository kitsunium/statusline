// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package collector

import (
	"context"
	"errors"
	"fmt"

	"github.com/kitsunium/statusline/collect/state"

	"github.com/kitsunium/statusline/ipc"
	"github.com/kitsunium/statusline/snapshot"
)

// execute never elevates and never retries a version that failed its probe.
func (h *CheckUpdateHandler) execute(ctx context.Context, in CheckUpdateInput) (CheckUpdateOutput, error) {
	// A development build has nothing to compare, and an opted-out one asks
	// nothing of the network
	if in.Disabled || ipc.CompareVersions(in.CurrentVersion, "v0.0.0") < 0 {
		return CheckUpdateOutput{}, nil
	}
	now, err := h.clock.Now(ctx)
	if err != nil {
		return CheckUpdateOutput{}, err
	}
	st, err := h.updateStore.LoadUpdate(ctx)
	if err != nil {
		return CheckUpdateOutput{}, fmt.Errorf("load update state: %w", err)
	}
	if !st.Due(now) {
		return CheckUpdateOutput{}, nil
	}
	st.CheckedAt = now
	rel, err := h.releaseSource.Latest(ctx)
	if err != nil || rel.Version == "" || rel.Version == st.BadVersion ||
		ipc.CompareVersions(rel.Version, in.CurrentVersion) <= 0 {
		return CheckUpdateOutput{}, errors.Join(err, h.updateStore.SaveUpdate(ctx, st))
	}
	notice := snapshot.UpdateNotice{Available: true, Version: rel.Version}
	// The release source probes the new binary and puts the previous one
	// back when the probe fails: that version is then never installed again
	if err := h.releaseSource.Install(ctx, rel); err != nil {
		st.Failures++
		if errors.Is(err, state.ErrProbeFailed) {
			st.BadVersion = rel.Version
		}
		return CheckUpdateOutput{}, errors.Join(fmt.Errorf("install %s: %w", rel.Version, err), h.updateStore.SaveUpdate(ctx, st))
	}
	st.Installed, st.Failures = rel.Version, 0
	return CheckUpdateOutput{Notice: notice, Installed: true}, h.updateStore.SaveUpdate(ctx, st)
}
