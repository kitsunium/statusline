package show

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"time"

	"github.com/kitsunium/statusline/ipc"
	"github.com/kitsunium/statusline/quota"
	"github.com/kitsunium/statusline/snapshot"
)

var (
	testNow = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	sgr     = regexp.MustCompile("\033\\[[0-9;]*m")
)

// fixedClock is a Clock at testNow.
type fixedClock struct{}

func (fixedClock) Now(context.Context) (time.Time, error) { return testNow, nil }

// source answers a fixed snapshot and records the key it was asked for.
type source struct {
	snap   snapshot.Snapshot
	origin string
	err    error
	key    ipc.Key
}

func (s *source) Snapshot(_ context.Context, key ipc.Key) (snapshot.Snapshot, string, error) {
	s.key = key
	return s.snap, s.origin, s.err
}

var errNoDaemon = errors.New("no daemon")

// warmSource is a daemon's snapshot whose API figures stdin must beat.
func warmSource() *source {
	api := quota.Set{
		Session: quota.NewLimit(quota.KindSession, "session", 99, testNow.Add(time.Hour), quota.SessionWindow, quota.SourceAPI),
		Scoped:  []quota.Limit{quota.NewLimit(quota.KindScoped, "opus", 48, testNow.Add(50*time.Hour), quota.WeeklyWindow, quota.SourceAPI)},
	}
	return &source{origin: OriginDaemon, snap: snapshot.Snapshot{WorkDir: "/work/elsewhere", Git: snapshot.GitStatus{Branch: "main"}, API: api}}
}

// warmStdin is a payload with its own session quota at 12 %.
func warmStdin() []byte {
	return []byte(`{"model":{"display_name":"Opus 5"},"session_id":"s","transcript_path":"/t.jsonl","workspace":{"current_dir":"/w"},` +
		`"rate_limits":{"five_hour":{"used_percentage":12,"resets_at":` + strconv.FormatInt(testNow.Add(2*time.Hour).Unix(), 10) + `}}}`)
}
