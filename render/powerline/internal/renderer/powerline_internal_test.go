package renderer

import (
	"strings"
	"testing"
	"time"

	"github.com/kitsunium/statusline/render/powerline/internal/model"
)

func TestPowerline_renderLine1(t *testing.T) {
	tests := []struct {
		name string
		data model.StatusLineData
	}{
		{
			name: "with git and changes",
			data: model.StatusLineData{
				Model:    model.ModelInfo{Name: "Opus", Version: "4.5"},
				Progress: model.Progress{Percent: 50},
				Icons:    model.IconConfig{OS: true, Model: true, Path: true, Git: true},
				Git:      model.GitStatus{Branch: "main", Modified: 1},
				System:   model.SystemInfo{OS: model.OSLinux},
				Dir:      "/workspace",
				Changes:  model.CodeChanges{Added: 10, Removed: 5},
			},
		},
		{
			name: "without git",
			data: model.StatusLineData{
				Model:  model.ModelInfo{Name: "Sonnet"},
				System: model.SystemInfo{OS: model.OSDarwin},
				Dir:    "/Users/test",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Powerline{}
			var sb strings.Builder
			r.renderLine1(&sb, tt.data)
			if sb.Len() == 0 {
				t.Error("renderLine1() produced empty output")
			}
		})
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		name  string
		input int
		want  string
	}{
		{name: "zero", input: 0, want: "0"},
		{name: "single digit", input: 5, want: "5"},
		{name: "multiple digits", input: 123, want: "123"},
		{name: "large number", input: 99999, want: "99999"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := itoa(tt.input); got != tt.want {
				t.Errorf("itoa(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestPowerline_renderOSSegment(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "renders OS segment"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Powerline{}
			var sb strings.Builder
			r.renderOSSegment(&sb, model.SystemInfo{OS: model.OSLinux}, true, model.HealthUnknown, nil, 0, BgBlue)
			if sb.Len() == 0 {
				t.Error("renderOSSegment() produced empty output")
			}
		})
	}
}

func TestPowerline_renderModelSegment(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "renders model segment"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Powerline{}
			var sb strings.Builder
			data := &ModelSegmentData{
				Model:    model.ModelInfo{Name: "Opus"},
				ShowIcon: true,
				Progress: model.Progress{Percent: 50},
				Cursor:   nil,
				NextBg:   BgBlue,
			}
			r.renderModelSegment(&sb, data)
			if sb.Len() == 0 {
				t.Error("renderModelSegment() produced empty output")
			}
		})
	}
}

func TestPowerline_renderPathSegment(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "renders path segment"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Powerline{}
			var sb strings.Builder
			r.renderPathSegment(&sb, "/workspace", true, true, "", 0)
			if sb.Len() == 0 {
				t.Error("renderPathSegment() produced empty output")
			}
		})
	}
}

func TestPowerline_renderGitSegment(t *testing.T) {
	tests := []struct {
		name string
		git  model.GitStatus
	}{
		{name: "with branch", git: model.GitStatus{Branch: "main"}},
		{name: "not in repo", git: model.GitStatus{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Powerline{}
			var sb strings.Builder
			r.renderGitSegment(&sb, tt.git, true, "", 0)
			_ = sb.String() // Just verify no panic
		})
	}
}

func TestPowerline_renderChangesSegment(t *testing.T) {
	tests := []struct {
		name    string
		changes model.CodeChanges
	}{
		{name: "with changes", changes: model.CodeChanges{Added: 10, Removed: 5}},
		{name: "no changes", changes: model.CodeChanges{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Powerline{}
			var sb strings.Builder
			r.renderChangesSegment(&sb, tt.changes)
			_ = sb.String() // Just verify no panic
		})
	}
}

func TestPowerline_renderLine2MCPBeforeUpdate(t *testing.T) {
	var sb strings.Builder
	(&Powerline{}).renderLine2(&sb, model.StatusLineData{
		Tasks:  model.TaskBoard{Epics: []model.Epic{sampleEpic()}},
		MCP:    model.MCPServers{{Name: "github", Enabled: true}, {Name: "tasks", Enabled: true}},
		Update: model.UpdateInfo{Available: true, Version: "v9.9.9"},
	})
	out := sb.String()
	epic, pill, update := strings.Index(out, "SDK status-line"), strings.Index(out, glyphs.MCP+" 2"), strings.Index(out, "v9.9.9")
	if epic < 0 || pill < 0 || update < 0 || !(epic < pill && pill < update) {
		t.Errorf("want epic, then MCP, then update; got %q", out)
	}
	if strings.Count(out, FgMCPEnabled+LeftRound) != 1 {
		t.Errorf("want a single MCP pill, got %q", out)
	}
}

func TestPowerline_renderUpdatePill(t *testing.T) {
	tests := []struct {
		name   string
		update model.UpdateInfo
	}{
		{name: "with update available", update: model.UpdateInfo{Available: true, Version: "v1.0.0"}},
		{name: "no update", update: model.UpdateInfo{Available: false}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Powerline{}
			var sb strings.Builder
			r.renderUpdatePill(&sb, tt.update)
			// Should produce output only if update available
			if tt.update.Available && sb.Len() == 0 {
				t.Error("renderUpdatePill() produced empty output for available update")
			}
		})
	}
}

func TestPowerline_renderModelSegmentCursorTakesTheInk(t *testing.T) {
	// The even-burn cursor used to be one orange for every model; on the pink
	// and lavender grounds it measured 2.2:1 and clashed in hue. It now takes
	// the pill's own ink, which is already held to the contrast floor.
	now := time.Now()
	for _, name := range []string{"Haiku 4.5", "Sonnet 5", "Opus 5", "Fable 5.1", "Mystery 1"} {
		t.Run(name, func(t *testing.T) {
			_, _, ink := GetModelColors(name)
			quota := model.Limit{
				Kind:     model.KindSession,
				Percent:  35,
				ResetsAt: now.Add(2 * time.Hour),
				Window:   5 * time.Hour,
			}
			r := &Powerline{}
			var sb strings.Builder
			r.renderModelSegment(&sb, &ModelSegmentData{
				Model:  model.ModelInfo{Name: name},
				Quotas: []model.Limit{quota},
				NextBg: BgBlue,
			})
			if !strings.Contains(sb.String(), ink+string(cursorChar)) {
				t.Errorf("cursor is not drawn in the model ink %q", ink)
			}
		})
	}
}

func TestPowerline_renderOSSegmentHealth(t *testing.T) {
	tests := []struct {
		name   string
		health model.ServiceHealth
		want   string
	}{
		{name: "operational", health: model.HealthOK, want: FgHealthOK + glyphs.Health},
		{name: "degraded", health: model.HealthDegraded, want: FgHealthDegraded + glyphs.Health},
		{name: "down", health: model.HealthDown, want: FgHealthDown + glyphs.Health},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sb strings.Builder
			(&Powerline{}).renderOSSegment(&sb, model.SystemInfo{OS: model.OSLinux}, true, tt.health, nil, 0, BgBlue)
			if !strings.Contains(sb.String(), tt.want) {
				t.Errorf("health glyph not drawn in its colour")
			}
		})
	}

	t.Run("unknown draws nothing", func(t *testing.T) {
		var sb strings.Builder
		(&Powerline{}).renderOSSegment(&sb, model.SystemInfo{OS: model.OSLinux}, true, model.HealthUnknown, nil, 0, BgBlue)
		if strings.Contains(sb.String(), glyphs.Health) {
			t.Errorf("an unknown state must not be drawn")
		}
	})
}

func TestPowerline_renderGitSegmentWorktrees(t *testing.T) {
	var with, without strings.Builder
	(&Powerline{}).renderGitSegment(&with, model.GitStatus{Branch: "main", Worktrees: 2}, true, "", 0)
	(&Powerline{}).renderGitSegment(&without, model.GitStatus{Branch: "main"}, true, "", 0)
	if !strings.Contains(with.String(), glyphs.Worktree+" 2") {
		t.Errorf("worktree count missing from %q", with.String())
	}
	if strings.Contains(without.String(), glyphs.Worktree) {
		t.Errorf("worktree glyph drawn without any linked worktree")
	}
}

// at is the render instant of a test: the pulse is on even seconds.
func at(unix int64) time.Time { return time.Unix(unix, 0) }

// sampleEpic is an epic whose tasks interleave every status.
func sampleEpic() model.Epic {
	return model.Epic{ID: 1, Title: "SDK status-line", Active: true, Tasks: model.TaskList{Items: []model.TaskItem{
		{ID: "1", Subject: "a rather long title that must never be shortened", Status: model.TaskInProgress},
		{ID: "2", Subject: "blocked", Status: model.TaskWaiting},
		{ID: "3", Subject: "later", Status: model.TaskPending},
		{ID: "4", Subject: "done late", Status: model.TaskCompleted},
		{ID: "5", Subject: "done late too", Status: model.TaskCompleted},
	}}}
}

func TestPowerline_renderLine2EpicsLeadTheLine(t *testing.T) {
	other := model.Epic{ID: 2, Title: "ktn-linter", Tasks: model.TaskList{Items: []model.TaskItem{
		{ID: "6", Subject: "x", Status: model.TaskPending},
	}}}
	mcp := model.MCPServers{{Name: "github", Enabled: true}}
	var sb strings.Builder
	(&Powerline{}).renderLine2(&sb, model.StatusLineData{
		Tasks: model.TaskBoard{Epics: []model.Epic{sampleEpic(), other}}, MCP: mcp,
	})
	out := sb.String()
	first, second, pill := strings.Index(out, "SDK status-line 2/5"), strings.Index(out, "ktn-linter 0/1"), strings.Index(out, glyphs.MCP+" 1")
	if first < 0 || second < 0 || pill < 0 || !(first < second && second < pill) {
		t.Errorf("want the epics in board order, then MCP; got %q", out)
	}
	if !strings.HasPrefix(out, " "+FgEpic+LeftRound+Reset+BgEpic+FgEpicInk+Bold+" SDK status-line") {
		t.Errorf("the pill opens with a mauve cap and the bold plum label, got %q", out)
	}
	if strings.Count(out, FgEpic+RightRound) != 2 {
		t.Errorf("each epic closes its own pill, got %q", out)
	}

	var none strings.Builder
	(&Powerline{}).renderLine2(&none, model.StatusLineData{MCP: mcp})
	if strings.Contains(none.String(), BgEpic) {
		t.Error("nothing open must draw no pill")
	}
}

func TestPowerline_renderEpicPillCollapsedOrExpanded(t *testing.T) {
	epic := sampleEpic()
	tests := []struct {
		name     string
		epic     model.Epic
		working  bool
		expanded bool
	}{
		{name: "active and working: expanded", epic: epic, working: true, expanded: true},
		{name: "active but idle: collapsed", epic: epic, working: false},
		{name: "working but not active: collapsed", epic: model.Epic{ID: 1, Title: epic.Title, Tasks: epic.Tasks}, working: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sb strings.Builder
			(&Powerline{}).renderLine2(&sb, model.StatusLineData{Tasks: model.TaskBoard{Epics: []model.Epic{tt.epic}}, Working: tt.working, Now: at(1_800_000_001)})
			out := sb.String()
			hasCells := strings.Contains(out, glyphs.TaskDone) || strings.Contains(out, glyphs.TaskOpen)
			hasHeadline := strings.Contains(out, "a rather long title")
			if hasCells != tt.expanded || hasHeadline != tt.expanded {
				t.Errorf("cells %v, headline %v, want both %v in %q", hasCells, hasHeadline, tt.expanded, out)
			}
			if !strings.Contains(out, "SDK status-line 2/5") {
				t.Errorf("title and count are always drawn, got %q", out)
			}
		})
	}
}

func TestPowerline_renderEpicCellsSortAndPulse(t *testing.T) {
	// Created in an order that interleaves the statuses: the cells still fill
	// from the left, done first, then under way, then the rest on the track
	bar := func(activeInk string) string {
		return BgEpic + " " + FgEpicInk + glyphs.TaskDone + glyphs.TaskDone + Reset +
			BgEpic + activeInk + glyphs.TaskDone + Reset +
			BgEpic + FgEpicTrack + glyphs.TaskOpen + glyphs.TaskOpen + Reset
	}

	var odd strings.Builder
	(&Powerline{}).renderEpicPill(&odd, sampleEpic(), true, at(1_800_000_001))
	if !strings.Contains(odd.String(), bar(FgEpicActive)) {
		t.Errorf("odd second: want the sorted cells in the plain amber, got %q", odd.String())
	}

	var even strings.Builder
	(&Powerline{}).renderEpicPill(&even, sampleEpic(), true, at(1_800_000_002))
	if !strings.Contains(even.String(), bar(FgEpicPulse)) {
		t.Errorf("even second: want the cell under way in the bright frame, got %q", even.String())
	}
	if odd.String() == even.String() {
		t.Error("two consecutive seconds must draw different frames")
	}
	if strings.Count(even.String(), FgEpicPulse) != 1 {
		t.Error("only the cell under way pulses")
	}
}

func TestPowerline_renderEpicPillAlwaysNamesATask(t *testing.T) {
	tests := []struct {
		name  string
		items []model.TaskItem
		want  string
	}{
		{name: "under way: its title", items: []model.TaskItem{
			{ID: "1", Subject: "doing", Status: model.TaskInProgress},
			{ID: "2", Subject: "blocked", Status: model.TaskWaiting},
		}, want: "doing"},
		{name: "nothing under way: the task waiting on the user", items: []model.TaskItem{
			{ID: "1", Subject: "next", Status: model.TaskPending},
			{ID: "2", Subject: "blocked", Status: model.TaskWaiting},
		}, want: "blocked"},
		{name: "nothing started: the next one", items: []model.TaskItem{
			{ID: "1", Subject: "done", Status: model.TaskCompleted},
			{ID: "2", Subject: "next", Status: model.TaskPending},
		}, want: "next"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sb strings.Builder
			(&Powerline{}).renderEpicPill(&sb, model.Epic{ID: 1, Title: "e", Active: true, Tasks: model.TaskList{Items: tt.items}}, true, at(1_800_000_001))
			// The headline sits inside the pill, in the regular plum ink
			want := BgEpic + FgEpicInk + " " + tt.want + Reset + BgEpic + " " + Reset + FgEpic + RightRound
			if !strings.Contains(sb.String(), want) {
				t.Errorf("got %q, want it to hold %q", sb.String(), want)
			}
		})
	}
}

func TestPowerline_renderEpicPillWaitingCellsOnTheTrack(t *testing.T) {
	list := model.TaskList{Items: []model.TaskItem{
		{ID: "1", Subject: "a", Status: model.TaskWaiting},
		{ID: "2", Subject: "b", Status: model.TaskPending},
		{ID: "3", Subject: "c", Status: model.TaskWaiting},
		{ID: "4", Subject: "d", Status: model.TaskCompleted},
	}}
	var sb strings.Builder
	(&Powerline{}).renderEpicPill(&sb, model.Epic{ID: 1, Title: "e", Active: true, Tasks: list}, true, at(1_800_000_001))
	want := BgEpic + FgEpicTrack + glyphs.TaskOpen + glyphs.TaskOpen + glyphs.TaskOpen + Reset
	if !strings.Contains(sb.String(), want) {
		t.Errorf("waiting cells are drawn like the ones to do, hollow on the track, got %q", sb.String())
	}
}

func TestPowerline_renderEpicPillWithoutTask(t *testing.T) {
	var sb strings.Builder
	(&Powerline{}).renderEpicPill(&sb, model.Epic{ID: 3, Title: "fresh", Active: true}, true, at(1_800_000_001))
	out := sb.String()
	if !strings.Contains(out, " fresh 0/0") || strings.Contains(out, glyphs.TaskDone) || strings.Contains(out, glyphs.TaskOpen) {
		t.Errorf("an epic focused before its first task shows 0/0 and no cell, got %q", out)
	}
	var nameless strings.Builder
	(&Powerline{}).renderEpicPill(&nameless, model.Epic{ID: 7}, false, at(1_800_000_001))
	if !strings.Contains(nameless.String(), " #7 0/0") {
		t.Errorf("a nameless epic is named by its id, got %q", nameless.String())
	}
}

func TestPowerline_renderEpicPillNoEpicIsTaches(t *testing.T) {
	list := model.TaskList{Items: []model.TaskItem{{ID: "1", Subject: "loose", Status: model.TaskInProgress}}}
	var sb strings.Builder
	(&Powerline{}).renderLine2(&sb, model.StatusLineData{
		Tasks:   model.TaskBoard{Epics: []model.Epic{{ID: model.NoEpic, Active: true, Tasks: list}}},
		Working: true,
	})
	out := sb.String()
	if !strings.Contains(out, " T\u00e2ches 0/1") || !strings.Contains(out, "loose") {
		t.Errorf("tasks under no epic get the T\u00e2ches pill, expanded while active, got %q", out)
	}
}

func TestPowerline_subagentsPlacement(t *testing.T) {
	epic := sampleEpic()
	epic.Subagents = 2
	data := model.StatusLineData{
		Model: model.ModelInfo{Name: "Opus"},
		Tasks: model.TaskBoard{Epics: []model.Epic{epic}, Unattributed: 3},
	}

	var line2 strings.Builder
	(&Powerline{}).renderLine2(&line2, data)
	want := BgEpic + FgEpicInk + Bold + " " + glyphs.Subagents + " 2" + Reset + BgEpic + " " + Reset + FgEpic + RightRound
	if !strings.Contains(line2.String(), want) {
		t.Errorf("the epic's subagents sit inside its pill, got %q", line2.String())
	}
	if strings.Contains(line2.String(), glyphs.Subagents+" 3") {
		t.Errorf("unattributed subagents do not belong on line 2, got %q", line2.String())
	}

	var line1 strings.Builder
	(&Powerline{}).renderOSSegment(&line1, model.SystemInfo{OS: model.OSLinux}, true, model.HealthOK, nil, 3, BgBlue)
	health, agents := strings.Index(line1.String(), glyphs.Health), strings.Index(line1.String(), BgWhite+FgBlack+Bold+glyphs.Subagents+" 3 ")
	if health < 0 || agents < 0 || agents < health {
		t.Errorf("unattributed subagents follow the health glyph in the OS segment, got %q", line1.String())
	}

	first, _, _ := strings.Cut((&Powerline{}).Render(data), "\n")
	if !strings.Contains(first, glyphs.Subagents+" 3 ") {
		t.Errorf("line 1 misses the unattributed subagents, got %q", first)
	}

	var none strings.Builder
	(&Powerline{}).renderOSSegment(&none, model.SystemInfo{OS: model.OSLinux}, true, model.HealthOK, nil, 0, BgBlue)
	if strings.Contains(none.String(), glyphs.Subagents) {
		t.Error("no unattributed subagent must draw nothing on line 1")
	}
}
