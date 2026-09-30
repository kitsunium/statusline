package collector

import (
	"context"
	"testing"

	"github.com/kitsunium/statusline/collect/state"
)

func givenSequenceRefreshNetwork(t *testing.T, rec sequenceRecorder) func(context.Context) error {
	t.Helper()
	e := &endpoints{token: "tok", usage: weekly(40), health: 1}
	uc := NewRefreshNetwork(&clock{now: t0}, rec.RecordCollectorNetworkStoreV1(&network{n: state.Network{}}),
		rec.RecordCollectorTokenStoreV1(e), rec.RecordCollectorUsageAPIV1(e), rec.RecordCollectorStatusPageV1(page{e}))
	return func(ctx context.Context) error {
		_, err := uc.Execute(ctx, RefreshNetworkInput{})
		return err
	}
}
