// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

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
