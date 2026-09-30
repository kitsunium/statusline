package renderer

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/kitsunium/statusline/render/powerline/internal/model"
)

// busyLine is a real-looking session: every quota with its window, a deep
// path, a long branch and changes. The directory avoids the home prefix so
// the result does not depend on who runs the test.
func busyLine(width int) model.StatusLineData {
	now := time.Now()
	week := 7 * 24 * time.Hour
	return model.StatusLineData{
		Model:  model.ModelInfo{Name: "Opus", Version: "5"},
		Effort: "xhigh",
		Limits: model.LimitSet{
			Context: model.NewLimit(model.KindContext, "context", 42, time.Time{}, 0, model.SourceStdin),
			Session: model.NewLimit(model.KindSession, "session", 34, now.Add(2*time.Hour), 5*time.Hour, model.SourceStdin),
			Weekly:  model.NewLimit(model.KindWeekly, "weekly", 61, now.Add(76*time.Hour), week, model.SourceStdin),
			Scoped:  []model.Limit{model.NewLimit(model.KindScoped, "opus", 48, now.Add(76*time.Hour), week, model.SourceAPI)},
		},
		Icons:    model.IconConfig{OS: true, Model: true, Path: true, Git: true},
		System:   model.SystemInfo{OS: model.OSLinux},
		Health:   model.HealthOK,
		Dir:      "/srv/code/worktrees/status-line-adaptive/internal/presentation/renderer",
		Git:      model.GitStatus{Branch: "feat/adaptive-line-condense-to-terminal-width", Modified: 3, Untracked: 1},
		Changes:  model.CodeChanges{Added: 12, Removed: 3},
		Terminal: model.TerminalInfo{Width: width},
	}
}

// line1At renders line one of data at one fit level, stripped.
func line1At(data model.StatusLineData, fit lineFit) string {
	var sb strings.Builder
	(&Powerline{}).renderLine1Fit(&sb, data, fit)
	return stripSGR(sb.String())
}

// describeSteps names each step in the order it is taken.
func describeSteps(steps []fitStep) []string {
	out := make([]string, 0, len(steps))
	for _, st := range steps {
		out = append(out, st.Describe())
	}
	return out
}

func TestFitStepsFollowTheUsersOrder(t *testing.T) {
	want := []string{
		"context: icon + %",
		"scoped: label + % + countdown",
		"weekly: label + % + countdown",
		"session: % + countdown",
		"path: 20 cells",
		"branch: 20 runes",
		"scoped: label + %",
		"weekly: label + %",
		"session: %",
		"path: last element",
		"branch: 12 runes",
		"scoped: initial + %",
		"weekly: initial + %",
		"changes: hidden",
		"path: hidden (inside a repository)",
		"model: name",
		"branch: 8 runes",
	}
	if got := describeSteps(fitSteps); !reflect.DeepEqual(got, want) {
		t.Errorf("steps =\n%q\nwant\n%q", got, want)
	}
	if len(fitLevels) != len(fitSteps)+1 || fitLevels[0] != (lineFit{}) {
		t.Errorf("fitLevels starts at the full line and adds one state per step")
	}
}

func TestFitPassesLowerEverySegmentOnceBeforeTwice(t *testing.T) {
	// With every weight inside the first pass, all first steps come before
	// any second step, and within a pass the lighter segment goes first
	var weights [segCount]int
	for id := range weights {
		weights[id] = 90 - 10*id
	}
	steps, levels := buildFitLevels(weights)
	for i := 1; i < len(steps); i++ {
		prev, cur := steps[i-1], steps[i]
		if cur.level < prev.level {
			t.Errorf("step %d (%s) is a lower pass than step %d (%s)", i, cur.Describe(), i-1, prev.Describe())
		}
		if cur.level == prev.level && weights[cur.seg] < weights[prev.seg] {
			t.Errorf("step %d (%s) should come before %s: lower weight", i, cur.Describe(), prev.Describe())
		}
	}
	if steps[0].seg != segModel {
		t.Errorf("the lightest segment shrinks first, got %s", steps[0].Describe())
	}
	// Each state differs from the previous one by exactly one level
	for i := 1; i < len(levels); i++ {
		diff := 0
		for id := range levels[i] {
			diff += levels[i][id] - levels[i-1][id]
		}
		if diff != 1 {
			t.Errorf("state %d moves %d levels, want 1", i, diff)
		}
	}
}

func TestWeightsFromEnv(t *testing.T) {
	def := weightsFromEnv("")
	for id, pol := range condensePolicy {
		if def[id] != pol.weight {
			t.Errorf("%s: default weight %d, want %d", pol.name, def[id], pol.weight)
		}
	}
	got := weightsFromEnv(" context=95 , weekly=5,bogus=1,path=-1,branch=1000,session=x,model,changes=7=8,scoped=0")
	want := def
	want[segContext], want[segWeekly], want[segScoped] = 95, 5, 0
	if got != want {
		t.Errorf("weightsFromEnv = %v, want %v (bad entries ignored)", got, want)
	}
}

func TestWeightOverrideReordersTheSteps(t *testing.T) {
	w := weightsFromEnv("context=65,weekly=5")
	steps, _ := buildFitLevels(w)
	got := describeSteps(steps)[:6]
	want := []string{
		"weekly: label + % + countdown",
		"scoped: label + % + countdown",
		"session: % + countdown",
		"path: 20 cells",
		"branch: 20 runes",
		"context: icon + %",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("first pass = %q, want %q", got, want)
	}
}

func TestSegmentLadders(t *testing.T) {
	data := busyLine(0)
	at := func(seg segID, level int) string {
		var fit lineFit
		fit[seg] = level
		return line1At(data, fit)
	}
	bar := func(line, after string) bool {
		// A bar sits right before the percentage: heavy or light rules
		i := strings.Index(line, after)
		return i > 0 && strings.ContainsAny(line[max(0, i-12):i], "\u2501\u2500")
	}
	tests := []struct {
		seg   segID
		level int
		check func(string) bool
		what  string
	}{
		{segContext, 0, func(l string) bool { return strings.Contains(l, nameContext) && bar(l, " 42%") }, "bar + label + %"},
		{segContext, 1, func(l string) bool {
			return !strings.Contains(l, nameContext) && strings.Contains(l, glyphs.Ctx+" 42%")
		}, "icon + %"},
		{segScoped, 0, func(l string) bool { return bar(l, " 48%") }, "bar"},
		{segScoped, 1, func(l string) bool { return strings.Contains(l, "Opus 48% "+glyphs.Reset) }, "label + % + countdown"},
		{segScoped, 2, func(l string) bool { return strings.Contains(l, "Opus 48% ") && strings.Count(l, glyphs.Reset) == 2 }, "label + %"},
		{segScoped, 3, func(l string) bool { return strings.Contains(l, " O 48%") }, "initial + %"},
		{segWeekly, 1, func(l string) bool { return strings.Contains(l, "Weekly 61% "+glyphs.Reset) }, "label + % + countdown"},
		{segWeekly, 2, func(l string) bool { return strings.Contains(l, "Weekly 61% ") && strings.Count(l, glyphs.Reset) == 2 }, "label + %"},
		{segWeekly, 3, func(l string) bool { return strings.Contains(l, " W 61%") }, "initial + %"},
		{segSession, 0, func(l string) bool { return bar(l, " 34%") }, "bar + % + countdown"},
		{segSession, 1, func(l string) bool { return strings.Contains(l, " 34% "+glyphs.Reset) && !bar(l, " 34%") }, "% + countdown"},
		{segSession, 2, func(l string) bool { return strings.Contains(l, " 34% ") && strings.Count(l, glyphs.Reset) == 2 }, "%"},
		{segPath, 0, func(l string) bool { return strings.Contains(l, "presentation/renderer") }, "default budget"},
		{segPath, 1, func(l string) bool { return strings.Contains(l, " .../renderer ") }, "20 cells"},
		{segPath, 3, func(l string) bool { return !strings.Contains(l, "renderer") && strings.Contains(l, "feat/") }, "hidden, branch kept"},
		{segBranch, 1, func(l string) bool { return strings.Contains(l, " feat/adaptive-line-\u2026 !3 ?1") }, "20 runes, counts kept"},
		{segBranch, 2, func(l string) bool { return strings.Contains(l, " feat/adapti\u2026 !3 ?1") }, "12 runes"},
		{segBranch, 3, func(l string) bool { return strings.Contains(l, " feat/ad\u2026 !3 ?1") }, "8 runes"},
		{segChanges, 1, func(l string) bool { return !strings.Contains(l, "+12") && !strings.Contains(l, "-3") }, "hidden"},
		{segModel, 1, func(l string) bool {
			return !strings.Contains(l, IconModel) && strings.Contains(l, " Opus 5 "+glyphs.EffortOn)
		}, "name, gauge kept"},
	}
	for _, tt := range tests {
		if line := at(tt.seg, tt.level); !tt.check(line) {
			t.Errorf("%s level %d: %s, got %q", condensePolicy[tt.seg].name, tt.level, tt.what, line)
		}
	}
	// No step along the default order widens the line (the bisection
	// relies on it); a step can be a no-op, as the path to its last element
	// when 20 cells already left only that
	first := VisibleWidth(line1At(data, fitLevels[0]))
	prev := first
	for i := 1; i < len(fitLevels); i++ {
		w := VisibleWidth(line1At(data, fitLevels[i]))
		if w > prev {
			t.Errorf("after %s the line is %d wide, wider than %d", fitSteps[i-1].Describe(), w, prev)
		}
		prev = w
	}
	if prev >= first-100 {
		t.Errorf("the whole ladder takes the busy line from %d to %d cells only", first, prev)
	}
}

func TestOSSegmentNeverShrinks(t *testing.T) {
	withMCPLine(t, false)
	data := busyLine(0)
	data.MCP = servers(7, 1)
	data.Tasks.Unattributed = 2
	os := func(fit lineFit) string {
		line := line1At(data, fit)
		head, _, _ := strings.Cut(line, SepRight)
		return head
	}
	full := os(fitLevels[0])
	if !strings.Contains(full, glyphs.MCP+" 7") || !strings.Contains(full, glyphs.Subagents+" 2") {
		t.Fatalf("the OS segment carries MCP and subagents, got %q", full)
	}
	for i, fit := range fitLevels {
		if got := os(fit); got != full {
			t.Errorf("state %d changes the OS segment: %q, want %q", i, got, full)
		}
	}
}

func TestCondensed(t *testing.T) {
	if got := Condensed(busyLine(1000)); len(got) != 0 {
		t.Errorf("a wide terminal condenses nothing, got %q", got)
	}
	light := busyLine(80)
	light.Limits.Scoped = nil
	for _, entry := range Condensed(light) {
		if strings.HasPrefix(entry, "scoped:") {
			t.Errorf("an absent segment is not reported, got %q", entry)
		}
	}
	got := Condensed(busyLine(160))
	if len(got) == 0 || got[0] != "context: icon + %" {
		t.Errorf("at 160 the context shrinks first, got %q", got)
	}
}

func TestFitLine1PicksTheFirstLevelThatFits(t *testing.T) {
	// Level n draws 100-5n cells
	render := func(sb *strings.Builder, fit lineFit) {
		for level, f := range fitLevels {
			if f == fit {
				sb.WriteString("\033[1m" + strings.Repeat("x", 100-5*level) + "\033[0m")
				return
			}
		}
	}
	tests := []struct {
		budget, level int
	}{
		{budget: 0, level: 0},
		{budget: 100, level: 0},
		{budget: 99, level: 1},
		{budget: 85, level: 3},
		{budget: 71, level: 6},
		{budget: 1, level: len(fitLevels) - 1},
	}
	for _, tt := range tests {
		line, level, fits := fitLine1(tt.budget, render)
		if want := tt.budget != 1; fits != want {
			t.Errorf("budget %d: fits = %v, want %v", tt.budget, fits, want)
		}
		if level != tt.level {
			t.Errorf("budget %d: level %d, want %d", tt.budget, level, tt.level)
		}
		if got := VisibleWidth(line); tt.budget > 0 && level < len(fitLevels)-1 && got > tt.budget {
			t.Errorf("budget %d: line is %d wide", tt.budget, got)
		}
	}
}

func TestLineBudget(t *testing.T) {
	for width, want := range map[int]int{0: 0, -3: 0, 80: 80 - lineMargin, 3: 1, 120: 120 - lineMargin} {
		if got := lineBudget(width); got != want {
			t.Errorf("lineBudget(%d) = %d, want %d", width, got, want)
		}
	}
}

func TestRenderLine1FitsTheTerminal(t *testing.T) {
	for _, width := range []int{200, 160, 120, 100, 80} {
		var sb strings.Builder
		(&Powerline{}).renderLine1(&sb, busyLine(width))
		if got := VisibleWidth(sb.String()); got > width-lineMargin {
			t.Errorf("COLUMNS=%d: line one is %d wide, over %d", width, got, width-lineMargin)
		}
	}
	// Without a width the line is drawn whole
	var sb strings.Builder
	(&Powerline{}).renderLine1(&sb, busyLine(0))
	if got, want := stripSGR(sb.String()), line1At(busyLine(0), lineFit{}); got != want {
		t.Errorf("no width: got %q, want the full line %q", got, want)
	}
	// Room enough keeps everything
	sb.Reset()
	(&Powerline{}).renderLine1(&sb, busyLine(1000))
	if !strings.Contains(stripSGR(sb.String()), nameContext) {
		t.Errorf("a wide terminal keeps the full line, got %q", stripSGR(sb.String()))
	}
}

func TestRenderLine1PathlessHandsOverToGit(t *testing.T) {
	var sb strings.Builder
	(&Powerline{}).renderLine1Fit(&sb, busyLine(0), fitLevels[len(fitLevels)-1])
	if !strings.Contains(sb.String(), BgGit+FgContext+SepRight) {
		t.Errorf("without the path the context hands over to git, got %q", sb.String())
	}
	// Outside a repository the path is never given up: it is all there is
	data := busyLine(0)
	data.Git = model.GitStatus{}
	if line := line1At(data, fitLevels[len(fitLevels)-1]); !strings.Contains(line, "renderer") {
		t.Errorf("no repository: the path stays, got %q", line)
	}
}

func TestLine2IsNeverCondensed(t *testing.T) {
	data := busyLine(40)
	data.Tasks = model.TaskBoard{Epics: []model.Epic{sampleEpic()}}
	data.Working = true
	out := (&Powerline{}).Render(data)
	if !strings.Contains(out, "a rather long title that must never be shortened") {
		t.Errorf("line two keeps full task titles at any width, got %q", out)
	}
}

func TestTruncateBranch(t *testing.T) {
	tests := []struct {
		branch string
		max    int
		want   string
	}{
		{"main", 0, "main"},
		{"main", 4, "main"},
		{"feature", 4, "fea…"},
		{"feature", 1, "…"},
		{"féature-été", 5, "féat…"},
		{"feat/adaptive-line-condense", 12, "feat/adapti…"},
	}
	for _, tt := range tests {
		if got := TruncateBranch(tt.branch, tt.max); got != tt.want {
			t.Errorf("TruncateBranch(%q, %d) = %q, want %q", tt.branch, tt.max, got, tt.want)
		}
	}
}

func TestLineFitQuotaLabel(t *testing.T) {
	weekly := model.Limit{Kind: model.KindWeekly}
	scoped := model.Limit{Kind: model.KindScoped, Label: "fable"}
	session := model.Limit{Kind: model.KindSession}
	var short lineFit
	short[segWeekly], short[segScoped] = 3, 3
	long := lineFit{}
	if got := long.quotaLabel(weekly); got != QuotaLabel(weekly) {
		t.Errorf("full weekly label = %q", got)
	}
	if got, want := short.quotaLabel(weekly), Labelled(glyphs.Quota, "W"); got != want {
		t.Errorf("short weekly label = %q, want %q", got, want)
	}
	if got, want := short.quotaLabel(scoped), Labelled(glyphs.Quota, "F"); got != want {
		t.Errorf("short scoped label = %q, want %q", got, want)
	}
	short[segSession] = 2
	if got := short.quotaLabel(session); got != QuotaLabel(session) {
		t.Errorf("session label is never shortened, got %q", got)
	}
	if got := short.quotaLabel(model.Limit{Kind: model.KindScoped}); got != QuotaLabel(model.Limit{Kind: model.KindScoped}) {
		t.Errorf("a nameless scoped quota keeps its label, got %q", got)
	}
}

func TestCompactLabelTextGlyphs(t *testing.T) {
	saved := glyphs
	t.Cleanup(func() { glyphs = saved })
	glyphs = textGlyphs
	ctx := model.Limit{Kind: model.KindContext}
	if got := compactLabel(ctx); got != nameContext {
		t.Errorf("without glyphs the context keeps its name, got %q", got)
	}
}

func BenchmarkRenderLine1(b *testing.B) {
	for _, bc := range []struct {
		name  string
		width int
	}{
		{name: "no-width", width: 0},
		{name: "fits-first", width: 1000},
		{name: "cols-120", width: 120},
		{name: "cols-100", width: 100},
		{name: "cols-80", width: 80},
		{name: "overflows-all", width: 20},
	} {
		data := busyLine(bc.width)
		// The MCP indicator sits in the OS segment at every level
		data.MCP = servers(7, 1)
		b.Run(bc.name, func(b *testing.B) {
			for b.Loop() {
				var sb strings.Builder
				(&Powerline{}).renderLine1(&sb, data)
			}
		})
	}
}
