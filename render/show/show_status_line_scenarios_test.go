// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package show

import (
	"strings"
	"testing"

	"github.com/kitsunium/statusline/ipc"
)

func givenShowStatusLineDrawsTheDaemonsSnapshotWithStdinWinning(t *testing.T) (ShowStatusLine, ShowStatusLineInput, func(*testing.T, ShowStatusLineOutput)) {
	t.Helper()
	src := warmSource()
	in := ShowStatusLineInput{Stdin: warmStdin(), Env: map[string]string{"COLUMNS": "200", "CLAUDE_CODE_TASK_LIST_ID": "team"}}
	return NewShowStatusLine(src, fixedClock{}), in, func(t *testing.T, out ShowStatusLineOutput) {
		if src.key != (ipc.Key{SessionID: "s", TranscriptPath: "/t.jsonl", SessionDir: "/w", TaskListID: "team"}) {
			t.Errorf("key = %+v", src.key)
		}
		plain := sgr.ReplaceAllString(out.Line, "")
		if !strings.Contains(plain, "12%") || strings.Contains(plain, "99%") {
			t.Errorf("stdin must win over the API: %q", plain)
		}
		if !strings.Contains(plain, "48%") || !strings.Contains(plain, "elsewhere") || !strings.Contains(plain, "main") {
			t.Errorf("the snapshot is not drawn: %q", plain)
		}
		if out.Origin != OriginDaemon {
			t.Errorf("origin = %q", out.Origin)
		}
	}
}

func givenShowStatusLineDrawsFromStdinAloneWhenNothingIsKnown(t *testing.T) (ShowStatusLine, ShowStatusLineInput, func(*testing.T, ShowStatusLineOutput)) {
	t.Helper()
	src := &source{origin: OriginNone, err: errNoDaemon}
	in := ShowStatusLineInput{Stdin: []byte(`{"model":{"display_name":"Haiku 4.5"},"workspace":{"current_dir":"/w"}}`)}
	return NewShowStatusLine(src, fixedClock{}), in, func(t *testing.T, out ShowStatusLineOutput) {
		plain := sgr.ReplaceAllString(out.Line, "")
		if !strings.Contains(plain, "Haiku 4.5") || !strings.Contains(plain, "/w") || out.Origin != OriginNone {
			t.Errorf("line = %q, origin %q", plain, out.Origin)
		}
	}
}
