package powerline_test

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"pgregory.net/rapid"

	"github.com/kitsunium/statusline/quota"
	"github.com/kitsunium/statusline/render/powerline"
	"github.com/kitsunium/statusline/render/session"
	"github.com/kitsunium/statusline/snapshot"
)

var (
	now = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	sgr = regexp.MustCompile("\033\\[[0-9;]*m")
)

// frameGen draws a plausible frame: every segment may be present or not.
func frameGen() *rapid.Generator[powerline.Frame] {
	return rapid.Custom(func(t *rapid.T) powerline.Frame {
		limit := func(kind quota.Kind, label string, window time.Duration) quota.Limit {
			if !rapid.Bool().Draw(t, "has "+label) {
				return quota.Limit{}
			}
			reset := now.Add(time.Duration(rapid.Int64Range(0, int64(window)).Draw(t, label+" reset")))
			return quota.NewLimit(kind, label, rapid.IntRange(0, 100).Draw(t, label+" %"), reset, window, quota.SourceStdin)
		}
		model := rapid.SampledFrom([]string{"Opus", "Sonnet", "Haiku", "Fable", "Mystery"}).Draw(t, "model")
		frame := powerline.Frame{
			Model:  session.ModelInfo{Name: model, Version: rapid.SampledFrom([]string{"", "5", "4.5 (1M context)"}).Draw(t, "version")},
			Effort: rapid.SampledFrom([]string{"", "low", "xhigh", "max", "ultra"}).Draw(t, "effort"),
			Limits: quota.Set{
				Context: quota.NewLimit(quota.KindContext, "ctx", rapid.IntRange(0, 100).Draw(t, "ctx"), time.Time{}, 0, quota.SourceStdin),
				Session: limit(quota.KindSession, "session", quota.SessionWindow),
				Weekly:  limit(quota.KindWeekly, "weekly", quota.WeeklyWindow),
			},
			Icons:   powerline.Icons{OS: true, Path: true, Git: true, Model: rapid.Bool().Draw(t, "model icon")},
			Dir:     "/" + strings.Repeat("deep/", rapid.IntRange(0, 8).Draw(t, "depth")) + "project",
			Changes: snapshot.Changes{Added: rapid.IntRange(0, 5000).Draw(t, "added"), Removed: rapid.IntRange(0, 5000).Draw(t, "removed")},
			System:  snapshot.System{OS: snapshot.OSLinux},
			Health:  snapshot.Health(rapid.IntRange(0, 3).Draw(t, "health")),
			Working: rapid.Bool().Draw(t, "working"),
			Now:     now,
		}
		if scoped := limit(quota.KindScoped, strings.ToLower(model), quota.WeeklyWindow); scoped.IsValid() {
			frame.Limits.Scoped = []quota.Limit{scoped}
		}
		if rapid.Bool().Draw(t, "in repo") {
			frame.Git = snapshot.GitStatus{
				Branch:    rapid.StringMatching(`[a-z][a-z/-]{0,60}`).Draw(t, "branch"),
				Modified:  rapid.IntRange(0, 50).Draw(t, "modified"),
				Untracked: rapid.IntRange(0, 50).Draw(t, "untracked"),
			}
		}
		for i := rapid.IntRange(0, 9).Draw(t, "servers"); i > 0; i-- {
			frame.MCP = append(frame.MCP, snapshot.MCPServer{Name: "s" + strconv.Itoa(i), Enabled: rapid.Bool().Draw(t, "on")})
		}
		if rapid.Bool().Draw(t, "epic") {
			frame.Tasks = snapshot.TaskBoard{Epics: []snapshot.Epic{{ID: 1, Title: "epic", Active: true, Tasks: snapshot.TaskList{Items: []snapshot.TaskItem{
				{ID: "1", Subject: "one", Status: snapshot.TaskCompleted}, {ID: "2", Subject: "two", Status: snapshot.TaskInProgress},
			}}}}}
		}
		return frame
	})
}

func lineOne(out string) string {
	line, _, _ := strings.Cut(out, "\n")
	return line
}

// TestPropertyTwoLines (render/property/two-lines).
func TestPropertyTwoLines(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		frame := frameGen().Draw(t, "frame")
		frame.Width = rapid.IntRange(1, 300).Draw(t, "width")
		out := powerline.Render(frame)
		if !strings.HasSuffix(out, "\n") || strings.Count(out, "\n") != 2 {
			t.Fatalf("Render() = %q, want two newline-terminated lines", out)
		}
	})
}

// TestPropertyLineOneFits (render/property/line-one-fits).
func TestPropertyLineOneFits(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		frame := frameGen().Draw(t, "frame")
		frame.Width = rapid.IntRange(1, 300).Draw(t, "width")
		line := lineOne(powerline.Render(frame))
		if powerline.VisibleWidth(line) <= frame.Width-4 {
			return
		}
		// Too wide is only allowed for the leanest line
		leanest := frame
		leanest.Width = 1
		if line != lineOne(powerline.Render(leanest)) {
			t.Fatalf("line one is %d cells for %d columns and is not the leanest", powerline.VisibleWidth(line), frame.Width)
		}
	})
}

// TestPropertyCondenseMonotonic (render/property/condense-monotonic).
func TestPropertyCondenseMonotonic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		frame := frameGen().Draw(t, "frame")
		wide := rapid.IntRange(2, 300).Draw(t, "wide")
		narrow := rapid.IntRange(1, wide).Draw(t, "narrow")
		w, n := frame, frame
		w.Width, n.Width = wide, narrow
		if powerline.VisibleWidth(lineOne(powerline.Render(n))) > powerline.VisibleWidth(lineOne(powerline.Render(w))) {
			t.Fatalf("narrowing %d -> %d widened line one", wide, narrow)
		}
		names := func(f powerline.Frame) map[string]bool {
			out := map[string]bool{}
			for _, c := range powerline.Condensed(f) {
				name, _, _ := strings.Cut(c, ":")
				out[name] = true
			}
			return out
		}
		atNarrow := names(n)
		for name := range names(w) {
			if !atNarrow[name] {
				t.Fatalf("segment %q condensed at %d is restored at %d", name, wide, narrow)
			}
		}
	})
}

// TestPropertyVisibleWidthEscapes (render/property/visible-width-ignores-escapes).
func TestPropertyVisibleWidthEscapes(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := rapid.StringMatching(`[ -~é│●━]{0,40}`).Draw(t, "s")
		at := rapid.IntRange(0, len([]rune(s))).Draw(t, "at")
		runes := []rune(s)
		code := strconv.Itoa(rapid.IntRange(0, 255).Draw(t, "code"))
		withEsc := string(runes[:at]) + "\033[38;5;" + code + "m" + string(runes[at:])
		if powerline.VisibleWidth(withEsc) != powerline.VisibleWidth(s) {
			t.Fatalf("an escape changed the width of %q", s)
		}
	})
}

// TestPropertyVisibleWidthAdditive (render/property/visible-width-additive).
func TestPropertyVisibleWidthAdditive(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := rapid.StringMatching(`[ -~é│●━\x{f17c}\x{4e2d}]{0,20}`).Draw(t, "a")
		b := rapid.StringMatching(`[ -~é│●━\x{f17c}\x{4e2d}]{0,20}`).Draw(t, "b")
		if powerline.VisibleWidth(a+b) != powerline.VisibleWidth(a)+powerline.VisibleWidth(b) {
			t.Fatalf("width(%q+%q) is not additive", a, b)
		}
	})
}

// TestPropertyParseWidth (render/property/parse-width-bounded).
func TestPropertyParseWidth(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		raw := rapid.OneOf(rapid.String(), rapid.Map(rapid.IntRange(-20000, 20000), strconv.Itoa)).Draw(t, "raw")
		w := powerline.ParseWidth(raw)
		if w < 1 || w > 10000 {
			t.Fatalf("ParseWidth(%q) = %d", raw, w)
		}
		if n, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil && n >= 1 && n <= 10000 && w != n {
			t.Fatalf("ParseWidth(%q) = %d, want %d", raw, w, n)
		}
	})
}

func TestParseWidthDefaults(t *testing.T) {
	for raw, want := range map[string]int{"": 120, "wide": 120, "0": 120, "-5": 120, "20000": 120, " 80 ": 80} {
		if got := powerline.ParseWidth(raw); got != want {
			t.Errorf("ParseWidth(%q) = %d, want %d", raw, got, want)
		}
	}
}

func TestIconsFromEnv(t *testing.T) {
	env := map[string]string{"STATUSLINE_ICON_OS": "false", "STATUSLINE_ICON_GIT": "No", "STATUSLINE_ICON_PATH": "0", "STATUSLINE_ICON_MODEL": "whatever"}
	got := powerline.IconsFromEnv(func(k string) string { return env[k] })
	if got != (powerline.Icons{Model: true}) {
		t.Errorf("IconsFromEnv() = %+v", got)
	}
	if all := powerline.IconsFromEnv(func(string) string { return "" }); all != (powerline.Icons{OS: true, Path: true, Git: true, Model: true}) {
		t.Errorf("default icons = %+v", all)
	}
}

func TestRenderDrawsAtTheFramesInstant(t *testing.T) {
	frame := powerline.Frame{
		Model:  session.ModelInfo{Name: "Opus", Version: "5"},
		Limits: quota.Set{Session: quota.NewLimit(quota.KindSession, "session", 30, now.Add(2*time.Hour+10*time.Minute+30*time.Second), quota.SessionWindow, quota.SourceStdin)},
		Width:  200,
		Now:    now,
	}
	plain := sgr.ReplaceAllString(powerline.Render(frame), "")
	if !strings.Contains(plain, "2h10") {
		t.Errorf("countdown not drawn from Frame.Now: %q", plain)
	}
	frame.Now = now.Add(time.Hour)
	if plain := sgr.ReplaceAllString(powerline.Render(frame), ""); !strings.Contains(plain, "1h10") {
		t.Errorf("countdown did not follow Frame.Now: %q", plain)
	}
}
