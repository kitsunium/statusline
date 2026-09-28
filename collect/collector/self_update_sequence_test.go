package collector

import (
	"context"
	"testing"
)

func givenSequenceSelfUpdate(t *testing.T, rec sequenceRecorder) func(context.Context) error {
	t.Helper()
	r := &releases{latest: "v1.3.0"}
	uc := NewCheckUpdate(&clock{now: t0}, rec.RecordCollectorUpdateStoreV1(r), rec.RecordCollectorReleaseSourceV1(r))
	return func(ctx context.Context) error {
		_, err := uc.Execute(ctx, CheckUpdateInput{CurrentVersion: "v1.2.0"})
		return err
	}
}
