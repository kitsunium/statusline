package show_test

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"pgregory.net/rapid"

	"github.com/kitsunium/statusline/ipc"
	"github.com/kitsunium/statusline/quota"
	"github.com/kitsunium/statusline/render/port"
	"github.com/kitsunium/statusline/render/show"
	"github.com/kitsunium/statusline/snapshot"
)

var (
	now = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	sgr = regexp.MustCompile("\033\\[[0-9;]*m")
)

// world records the port calls in order and answers with a snapshot.
type world struct {
	calls  []string
	key    ipc.Key
	snap   snapshot.Snapshot
	origin port.SnapshotOrigin
}

func (w *world) Now() time.Time {
	w.calls = append(w.calls, "clock")
	return now
}

func (w *world) Snapshot(_ context.Context, key ipc.Key) (snapshot.Snapshot, port.SnapshotOrigin) {
	w.calls = append(w.calls, "snapshot")
	w.key = key
	return w.snap, w.origin
}

func run(w *world, stdin string, env map[string]string) show.ShowOutput {
	out, _ := show.New(show.Deps{Clock: w, Snapshots: w}).Execute(context.Background(), show.ShowInput{
		Stdin: []byte(stdin), Columns: env["COLUMNS"], TaskListID: env["CLAUDE_CODE_TASK_LIST_ID"],
		Getenv: func(k string) string { return env[k] },
	})
	return out
}

// TestSequenceRenderWarm pins the client side of design/sequences/render-warm.yaml:
// the key is asked, then the frame is drawn at the clock's instant, and stdin
// wins over the daemon's quotas.
func TestSequenceRenderWarm(t *testing.T) {
	api := quota.Set{
		Session: quota.NewLimit(quota.KindSession, "session", 99, now.Add(time.Hour), quota.SessionWindow, quota.SourceAPI),
		Scoped:  []quota.Limit{quota.NewLimit(quota.KindScoped, "opus", 48, now.Add(50*time.Hour), quota.WeeklyWindow, quota.SourceAPI)},
	}
	w := &world{origin: port.OriginDaemon, snap: snapshot.Snapshot{WorkDir: "/work/elsewhere", Git: snapshot.GitStatus{Branch: "main"}, API: api}}
	stdin := `{"model":{"display_name":"Opus 5"},"session_id":"s","transcript_path":"/t.jsonl","workspace":{"current_dir":"/w"},` +
		`"rate_limits":{"five_hour":{"used_percentage":12,"resets_at":` + itoa(now.Add(2*time.Hour).Unix()) + `}}}`
	out := run(w, stdin, map[string]string{"COLUMNS": "200", "CLAUDE_CODE_TASK_LIST_ID": "team"})
	if got := strings.Join(w.calls, ","); got != "snapshot,clock" {
		t.Errorf("calls = %s, want snapshot then clock", got)
	}
	if w.key != (ipc.Key{SessionID: "s", TranscriptPath: "/t.jsonl", SessionDir: "/w", TaskListID: "team"}) {
		t.Errorf("key = %+v", w.key)
	}
	plain := sgr.ReplaceAllString(out.Line, "")
	if !strings.Contains(plain, "12%") || strings.Contains(plain, "99%") {
		t.Errorf("stdin must win over the API: %q", plain)
	}
	if !strings.Contains(plain, "48%") || !strings.Contains(plain, "elsewhere") || !strings.Contains(plain, "main") {
		t.Errorf("the snapshot is not drawn: %q", plain)
	}
	if out.Origin != port.OriginDaemon {
		t.Errorf("origin = %v", out.Origin)
	}
}

func TestNothingKnownRendersFromStdin(t *testing.T) {
	w := &world{origin: port.OriginNone}
	out := run(w, `{"model":{"display_name":"Haiku 4.5"},"workspace":{"current_dir":"/w"}}`, nil)
	plain := sgr.ReplaceAllString(out.Line, "")
	if !strings.Contains(plain, "Haiku 4.5") || !strings.Contains(plain, "/w") {
		t.Errorf("line = %q", plain)
	}
}

// TestPropertyShowTotal (render/property/show-total).
func TestPropertyShowTotal(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		stdin := rapid.SliceOf(rapid.Byte()).Draw(t, "stdin")
		cols := rapid.String().Draw(t, "cols")
		out := run(&world{origin: port.OriginNone}, string(stdin), map[string]string{"COLUMNS": cols})
		if strings.Count(out.Line, "\n") != 2 || !strings.HasSuffix(out.Line, "\n") {
			t.Fatalf("ShowStatusLine(%q) = %q", stdin, out.Line)
		}
	})
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
