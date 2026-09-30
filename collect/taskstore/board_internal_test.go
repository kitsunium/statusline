package taskstore

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kitsunium/statusline/snapshot"
)

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestNewBoardLocatesTheStores(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv(configDirEnv, cfg)
	p := newBoard("896dc174-db17-4ec1", "", time.Now())
	if want := filepath.Join(cfg, "kodflow", "sessions", "896dc174-db17-4ec1"); p.sessionDir != want {
		t.Errorf("sessionDir = %q, want %q", p.sessionDir, want)
	}
	if want := filepath.Join(cfg, "tasks", "session-896dc174"); p.builtinDir != want {
		t.Errorf("builtinDir = %q, want %q", p.builtinDir, want)
	}
	if got, want := newBoard("896dc174", "my team/list", time.Now()).builtinDir, filepath.Join(cfg, "tasks", "my-team-list"); got != want {
		t.Errorf("builtinDir with override = %q, want %q", got, want)
	}
	if p := newBoard("", "", time.Now()); p.sessionDir != "" || p.builtinDir != "" {
		t.Errorf("provider without session located %q / %q", p.sessionDir, p.builtinDir)
	}
}

// titles lists the board's epics as "title done/total", in display order.
func titles(board snapshot.TaskBoard) []string {
	out := make([]string, 0, len(board.Epics))
	for _, epic := range board.Epics {
		out = append(out, epic.Title+" "+strconv.Itoa(epic.Tasks.Done())+"/"+strconv.Itoa(epic.Tasks.Total()))
	}
	return out
}

func TestBoardFallsBackToTheBuiltinListOnlyWithoutTheMCPFile(t *testing.T) {
	dir := t.TempDir()
	p := &sessionBoard{sessionDir: filepath.Join(dir, "s"), builtinDir: filepath.Join(dir, "b"), now: time.Now}
	writeFile(t, filepath.Join(p.builtinDir, "1.json"), `{"id":"1","subject":"builtin","status":"pending"}`)

	board := p.board()
	if len(board.Epics) != 1 || board.Epics[0].ID != snapshot.NoEpic || !board.Epics[0].Active || board.Epics[0].Tasks.Items[0].Subject != "builtin" {
		t.Fatalf("without an MCP file the built-in list is the active no-epic pill, got %+v", board)
	}

	writeFile(t, filepath.Join(p.sessionDir, "tasks.json"), `{"version":2,"epics":[],"active":{},"tasks":[
		{"id":"1","agent":"a1b2","subject":"sub","status":"pending"}]}`)
	if got := p.board(); len(got.Epics) != 0 {
		t.Errorf("an MCP file with nothing open for main draws nothing, got %+v", got)
	}
	writeFile(t, filepath.Join(p.sessionDir, "tasks.json"), `{"tasks":`)
	if got := p.board(); len(got.Epics) != 0 {
		t.Errorf("a malformed MCP file draws nothing, got %+v", got)
	}
}

func TestBoardReadsVersion2(t *testing.T) {
	dir := t.TempDir()
	p := &sessionBoard{sessionDir: dir, now: time.Now}
	writeFile(t, filepath.Join(dir, "tasks.json"), `{"version":2,"next_id":12,"next_epic":6,
		"epics":[
			{"id":1,"agent":"main","title":"old","created":100,"touched":900},
			{"id":2,"agent":"main","title":"SDK","created":200,"touched":300},
			{"id":3,"agent":"main","title":"ktn","created":300,"touched":800},
			{"id":4,"agent":"main","title":"done","created":400,"touched":950},
			{"id":5,"agent":"a1b2","title":"theirs","created":500,"touched":999}],
		"active":{"main":2,"a1b2":5},
		"tasks":[
			{"id":"1","agent":"main","epic":1,"subject":"o","status":"pending","created":100,"updated":900},
			{"id":"10","agent":"main","epic":2,"subject":"ten","status":"pending"},
			{"id":"3","agent":"main","epic":2,"subject":"three","status":"in_progress"},
			{"id":"4","agent":"main","epic":3,"subject":"k","status":"waiting"},
			{"id":"5","agent":"main","epic":4,"subject":"d","status":"completed"},
			{"id":"6","agent":"main","epic":0,"subject":"loose","status":"pending","created":850,"updated":850},
			{"id":"7","agent":"a1b2","epic":5,"subject":"sub","status":"pending"}]}`)
	board := p.board()
	want := []string{"SDK 0/2", "old 0/1", " 0/1", "ktn 0/1"}
	if got := titles(board); strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("epics = %q, want active first then by touched desc: %q", got, want)
	}
	sdk := board.Epics[0]
	if !sdk.Active || sdk.Tasks.Items[0].ID != "3" || sdk.Tasks.Items[1].ID != "10" {
		t.Errorf("active epic = %+v, want active with tasks 3 then 10", sdk)
	}
	if board.Epics[2].ID != snapshot.NoEpic || board.Epics[2].Active {
		t.Errorf("tasks under no epic are an inactive pill while an epic is active, got %+v", board.Epics[2])
	}
}

func TestBoardOpenEpics(t *testing.T) {
	dir := t.TempDir()
	p := &sessionBoard{sessionDir: dir, now: time.Now}
	writeFile(t, filepath.Join(dir, "tasks.json"), `{"version":2,
		"epics":[{"id":1,"agent":"main","title":"fresh","touched":1},{"id":2,"agent":"main","title":"idle","touched":2}],
		"active":{"main":1},"tasks":[{"id":"1","agent":"main","epic":0,"subject":"x","status":"pending"}]}`)
	board := p.board()
	if got := titles(board); strings.Join(got, "|") != "fresh 0/0| 0/1" {
		t.Errorf("an active empty epic is open, an inactive empty one is not; got %q", got)
	}

	writeFile(t, filepath.Join(dir, "tasks.json"), `{"version":2,
		"epics":[{"id":1,"agent":"main","title":"gone","touched":1}],
		"active":{"main":9},"tasks":[
			{"id":"1","agent":"main","epic":0,"subject":"x","status":"pending"},
			{"id":"2","agent":"main","epic":7,"subject":"orphan","status":"pending"}]}`)
	board = p.board()
	if len(board.Epics) != 1 || board.Epics[0].ID != snapshot.NoEpic || !board.Epics[0].Active || board.Epics[0].Tasks.Total() != 1 {
		t.Errorf("with no valid active epic the no-epic pill is active, orphans left out; got %+v", board.Epics)
	}
}

func TestBoardActiveEpicEdgeCases(t *testing.T) {
	dir := t.TempDir()
	p := &sessionBoard{sessionDir: dir, now: time.Now}
	const epics = `"epics":[{"id":1,"title":"one","touched":5},{"id":2,"agent":"main","title":"two","touched":1}]`
	const tasks = `"tasks":[
		{"id":"1","agent":"main","epic":1,"subject":"a","status":"completed"},
		{"id":"2","agent":"main","epic":2,"subject":"b","status":"completed"},
		{"id":"3","agent":"main","epic":2,"subject":"rework","status":"pending"},
		{"id":"4","agent":"main","epic":0,"subject":"loose","status":"pending"}]`
	tests := []struct {
		name   string
		active string
		want   string
		first  bool
	}{
		{name: "active missing: T\u00e2ches is active", active: `{}`, want: " 0/1|two 1/2", first: true},
		{name: "active 0: T\u00e2ches is active", active: `{"main":0}`, want: " 0/1|two 1/2", first: true},
		{name: "active names no epic: T\u00e2ches is active", active: `{"main":42}`, want: " 0/1|two 1/2", first: true},
		{name: "active epic all done: closed, nothing drawn for it", active: `{"main":1}`, want: "two 1/2| 0/1", first: false},
		{name: "reopened epic, focused: expanded candidate", active: `{"main":2}`, want: "two 1/2| 0/1", first: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writeFile(t, filepath.Join(dir, "tasks.json"), `{"version":2,`+epics+`,"active":`+tt.active+`,`+tasks+`}`)
			board := p.board()
			if got := strings.Join(titles(board), "|"); got != tt.want {
				t.Fatalf("epics = %q, want %q", got, tt.want)
			}
			if board.Epics[0].Active != tt.first {
				t.Errorf("first pill active = %v, want %v", board.Epics[0].Active, tt.first)
			}
			for _, epic := range board.Epics[1:] {
				if epic.Active {
					t.Errorf("only the first pill may be active, got %+v", epic)
				}
			}
		})
	}
}

func TestBoardMigratesVersion1(t *testing.T) {
	dir := t.TempDir()
	p := &sessionBoard{sessionDir: dir, now: time.Now}
	writeFile(t, filepath.Join(dir, "tasks.json"), `{"epics":{"main":{"id":2,"title":"SDK"},"a1b2":{"id":3,"title":"theirs"}},"tasks":[
		{"id":"1","agent":"main","epic":1,"subject":"old subject","status":"completed"},
		{"id":"2","agent":"main","epic":2,"subject":"freeze goldens","status":"in_progress"},
		{"id":"3","agent":"main","epic":2,"subject":"write the SDK","status":"pending"},
		{"id":"4","agent":"main","subject":"before epics","status":"pending"}]}`)
	board := p.board()
	if got := titles(board); strings.Join(got, "|") != "SDK 0/2| 0/1" {
		t.Fatalf("v1: the current epic is active, tasks without epic are no-epic; got %q", got)
	}
	if !board.Epics[0].Active || board.Epics[0].Tasks.Current() != "freeze goldens" {
		t.Errorf("v1 current epic = %+v", board.Epics[0])
	}

	writeFile(t, filepath.Join(dir, "tasks.json"), `{"tasks":[{"id":"1","agent":"main","subject":"before epics","status":"pending"}]}`)
	board = p.board()
	if len(board.Epics) != 1 || board.Epics[0].ID != snapshot.NoEpic || !board.Epics[0].Active {
		t.Errorf("a file without epics is one active no-epic pill, got %+v", board.Epics)
	}
}

func TestBuiltinTasks(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "10.json"), `{"id":"10","subject":"ten","status":"pending"}`)
	writeFile(t, filepath.Join(dir, "2.json"), `{"id":"2","subject":"two","status":"in_progress"}`)
	writeFile(t, filepath.Join(dir, "1.json"), `{"id":"1","subject":"one","status":"completed"}`)
	writeFile(t, filepath.Join(dir, "3.json"), `{"id":`)
	writeFile(t, filepath.Join(dir, ".lock"), ``)

	list := (&sessionBoard{builtinDir: dir}).builtinTasks()
	if list.Total() != 3 {
		t.Fatalf("Total() = %d, want 3 (malformed file skipped)", list.Total())
	}
	if list.Items[0].ID != "1" || list.Items[1].ID != "2" || list.Items[2].ID != "10" {
		t.Errorf("order = %v, want numeric id order", list.Items)
	}
	if got := (&sessionBoard{}).builtinTasks(); got.Total() != 0 {
		t.Errorf("unlocated provider returned %d tasks", got.Total())
	}
}

func TestBoardAttributesSubagents(t *testing.T) {
	dir := t.TempDir()
	now := time.Unix(1_800_000_000, 0)
	p := &sessionBoard{sessionDir: dir, now: func() time.Time { return now }}
	writeFile(t, filepath.Join(dir, "tasks.json"), `{"version":2,
		"epics":[{"id":1,"agent":"main","title":"one","touched":2},{"id":2,"agent":"main","title":"two","touched":1},
			{"id":3,"agent":"main","title":"closed","touched":3}],
		"active":{"main":1},"tasks":[
			{"id":"1","agent":"main","epic":1,"subject":"a","status":"pending"},
			{"id":"2","agent":"main","epic":2,"subject":"b","status":"pending"},
			{"id":"3","agent":"main","epic":3,"subject":"c","status":"completed"},
			{"id":"4","agent":"main","epic":0,"subject":"d","status":"pending"}]}`)
	if got := p.board(); got.Unattributed != 0 || got.Epics[0].Subagents != 0 {
		t.Errorf("no registry: %+v, want no subagent", got)
	}

	recent, old := now.Add(-time.Minute).Unix(), now.Add(-13*time.Hour).Unix()
	writeFile(t, filepath.Join(dir, "agents.json"), `{"agents":{
		"a":{"type":"Explore","started":`+itoa(recent)+`,"stopped":null,"epic":1},
		"b":{"type":"Plan","started":`+itoa(recent)+`,"stopped":`+itoa(recent)+`,"epic":1},
		"c":{"type":"Explore","started":`+itoa(recent)+`,"stopped":null,"epic":1},
		"d":{"type":"Explore","started":`+itoa(old)+`,"stopped":null,"epic":2},
		"e":{"type":"Explore","started":`+itoa(recent)+`,"stopped":null,"epic":2},
		"f":{"type":"Explore","started":`+itoa(recent)+`,"stopped":null,"epic":0},
		"g":{"type":"Explore","started":`+itoa(recent)+`,"stopped":null,"epic":3},
		"h":{"type":"Explore","started":`+itoa(recent)+`,"stopped":null}}}`)
	board := p.board()
	if board.Epics[0].Subagents != 2 || board.Epics[1].Subagents != 1 {
		t.Errorf("epic counts = %d, %d; want 2 (stopped excluded), 1 (stale excluded)", board.Epics[0].Subagents, board.Epics[1].Subagents)
	}
	if board.Epics[2].ID != snapshot.NoEpic || board.Epics[2].Subagents != 0 {
		t.Errorf("the no-epic pill takes no subagent, got %+v", board.Epics[2])
	}
	if board.Unattributed != 3 {
		t.Errorf("Unattributed = %d, want 3 (no epic, closed epic, missing epic)", board.Unattributed)
	}

	writeFile(t, filepath.Join(dir, "agents.json"), `{"agents":`)
	if got := p.board(); got.Unattributed != 0 || got.Epics[0].Subagents != 0 {
		t.Errorf("malformed registry: %+v, want no subagent", got)
	}
	if got := (&sessionBoard{}).board(); !got.IsEmpty() {
		t.Errorf("unlocated provider: %+v, want an empty board", got)
	}
}

func TestBoardSubagentsWithTheBuiltinList(t *testing.T) {
	dir := t.TempDir()
	now := time.Unix(1_800_000_000, 0)
	p := &sessionBoard{sessionDir: filepath.Join(dir, "s"), builtinDir: filepath.Join(dir, "b"), now: func() time.Time { return now }}
	writeFile(t, filepath.Join(p.builtinDir, "1.json"), `{"id":"1","subject":"builtin","status":"pending"}`)
	writeFile(t, filepath.Join(p.sessionDir, "agents.json"), `{"agents":{"a":{"started":`+itoa(now.Unix())+`,"stopped":null,"epic":0}}}`)
	if got := p.board(); got.Unattributed != 1 || got.Epics[0].Subagents != 0 {
		t.Errorf("subagents beside the built-in list go to line 1, got %+v", got)
	}
}

func itoa(v int64) string {
	return strconv.FormatInt(v, 10)
}
