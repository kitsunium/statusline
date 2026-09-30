package show

import (
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// propertyShowStatusLineTotal: two lines for any stdin bytes and any
// COLUMNS, whether the snapshot source answers or not.
func propertyShowStatusLineTotal(t *testing.T, seed uint64) {
	t.Logf("design seed %d (rapid v1.2 takes its own seed from -rapid.seed)", seed)
	rapid.Check(t, func(t *rapid.T) {
		stdin := rapid.SliceOf(rapid.Byte()).Draw(t, "stdin")
		cols := rapid.String().Draw(t, "cols")
		src := &source{origin: OriginNone}
		if rapid.Bool().Draw(t, "fails") {
			src.err = errNoDaemon
		}
		out, err := NewShowStatusLine(src, fixedClock{}).Execute(t.Context(), ShowStatusLineInput{Stdin: stdin, Env: map[string]string{"COLUMNS": cols}})
		if err != nil || strings.Count(out.Line, "\n") != 2 || !strings.HasSuffix(out.Line, "\n") {
			t.Fatalf("ShowStatusLine(%q) = %q, %v", stdin, out.Line, err)
		}
	})
}
