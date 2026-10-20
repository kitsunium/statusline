// Written once from the design; this file is yours. The shells in
// gen_design.go delegate to what it holds; a body that still answers
// errNotImplemented is a stub.

package show

import (
	"context"
	"testing"
)

func givenSequenceRenderWarm(t *testing.T, rec sequenceRecorder) func(context.Context) error {
	t.Helper()
	uc := NewShowStatusLine(rec.RecordShowSnapshotSourceV1(warmSource()), rec.RecordShowClockV1(fixedClock{}))
	return func(ctx context.Context) error {
		_, err := uc.Execute(ctx, ShowStatusLineInput{Stdin: warmStdin()})
		return err
	}
}
