// Package tasks reads the session's epics, tasks and running subagents.
package taskstore

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/kitsunium/statusline/snapshot"
)

// Task list location constants.
const (
	// configDirEnv relocates the whole configuration directory.
	configDirEnv string = "CLAUDE_CONFIG_DIR"
	// sessionPrefix and sessionIDLen form the default built-in list id: the
	// first characters of the session id.
	sessionPrefix string = "session-"
	sessionIDLen  int    = 8
	// mainAgent owns the tasks the main conversation creates.
	mainAgent string = "main"
	// staleAgent is how long a subagent may run without a stop event before it
	// is taken for one whose stop was lost with a crashed session.
	staleAgent time.Duration = 12 * time.Hour
)

// unsafeID matches the characters replaced in a list or session directory name.
var unsafeID = regexp.MustCompile(`[^a-zA-Z0-9_-]`)

// board reads the task list of one session from two places.
//
// The kodflow tasks MCP (kodflow-hooks) keeps every agent's tasks in
// <config>/kodflow/sessions/<session>/tasks.json, next to agents.json, the
// running subagents recorded by the SubagentStart/SubagentStop hooks. It is
// preferred: it tells the main agent's tasks apart. The built-in task tools'
// one-file-per-task list under <config>/tasks/<list>/ is the fallback.
type board struct {
	sessionDir string
	builtinDir string
	now        func() time.Time
}

// taskFile is one task, in either store.
type taskFile struct {
	ID      string `json:"id"`
	Agent   string `json:"agent"`
	Subject string `json:"subject"`
	Status  string `json:"status"`
	Epic    int    `json:"epic"`
	Created int64  `json:"created"`
	Updated int64  `json:"updated"`
}

// epicFile is one epic of the tasks MCP file.
type epicFile struct {
	ID      int    `json:"id"`
	Agent   string `json:"agent"`
	Title   string `json:"title"`
	Touched int64  `json:"touched"`
}

// sessionTasks is the tasks MCP file.
type sessionTasks struct {
	Tasks []taskFile `json:"tasks"`
	// Epics is a list since version 2, a map of each agent's current epic in
	// version 1: it is decoded by readEpics.
	Epics json.RawMessage `json:"epics"`
	// Active maps each agent to the epic it is focused on (version 2).
	Active map[string]int `json:"active"`
}

// sessionAgents is the running-agents file.
type sessionAgents struct {
	Agents map[string]struct {
		Started int64  `json:"started"`
		Stopped *int64 `json:"stopped"`
		// Epic is the main agent's active epic when the subagent started.
		Epic int `json:"epic"`
	} `json:"agents"`
}

// openEpic is an epic to show, with the time used to order it.
type openEpic struct {
	epic    snapshot.Epic
	touched int64
}

// newBoard locates the task stores of a session.
//
// Params:
//   - sessionID: session identifier from stdin, may be empty
//   - listID: CLAUDE_CODE_TASK_LIST_ID of the client, may be empty
//   - now: the instant subagents are judged stale against
//
// Returns:
//   - *board: reader for that session, reading nothing without an id
func newBoard(sessionID, listID string, now time.Time) *board {
	clock := func() time.Time { return now }
	base := os.Getenv(configDirEnv)
	// Default to the per-user configuration directory
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return &board{now: clock}
		}
		base = filepath.Join(home, ".claude")
	}
	p := &board{now: clock}
	// The kodflow stores are keyed by the whole session id
	if sessionID != "" {
		p.sessionDir = filepath.Join(base, "kodflow", "sessions", unsafeID.ReplaceAllString(sessionID, "-"))
	}
	// The built-in list falls back to a per-session id
	if listID == "" && sessionID != "" {
		listID = sessionPrefix + sessionID[:min(sessionIDLen, len(sessionID))]
	}
	if listID != "" {
		p.builtinDir = filepath.Join(base, "tasks", unsafeID.ReplaceAllString(listID, "-"))
	}
	return p
}

// board returns the main agent's open epics and the running subagents.
//
// The tasks MCP file wins whenever it exists; without it the built-in list is
// drawn as the tasks filed under no epic.
//
// Returns:
//   - snapshot.TaskBoard: open epics in display order, subagents attributed
func (p *board) board() snapshot.TaskBoard {
	epics, found := p.mcpEpics()
	// No MCP file: the built-in list, if it is still open, is the only pill
	if !found {
		if list := p.builtinTasks(); list.IsActive() {
			epics = []snapshot.Epic{{ID: snapshot.NoEpic, Active: true, Tasks: list}}
		}
	}
	board := snapshot.TaskBoard{Epics: epics}
	p.attributeSubagents(&board)
	return board
}

// attributeSubagents counts the running subagents into the epic each was
// started for, and those whose epic is not shown into Unattributed.
//
// Params:
//   - board: board whose epics receive the counts
func (p *board) attributeSubagents(board *snapshot.TaskBoard) {
	// No session located, nothing to count
	if p.sessionDir == "" {
		return
	}
	data, err := os.ReadFile(filepath.Join(p.sessionDir, "agents.json"))
	// No registry yet means no subagent has started
	if err != nil {
		return
	}
	var reg sessionAgents
	// A registry mid-write reads as empty until the next redraw
	if err := json.Unmarshal(data, &reg); err != nil {
		return
	}
	shown := make(map[int]int, len(board.Epics))
	// Only real epics take subagents: those started with none go to line 1
	for idx, epic := range board.Epics {
		if epic.ID != snapshot.NoEpic {
			shown[epic.ID] = idx
		}
	}
	cutoff := p.now().Add(-staleAgent).Unix()
	// Count the agents still running, forgetting any whose stop was lost
	for _, agent := range reg.Agents {
		if agent.Stopped != nil || agent.Started < cutoff {
			continue
		}
		// The epic's own pill when it is drawn, the OS segment otherwise
		if idx, ok := shown[agent.Epic]; ok {
			board.Epics[idx].Subagents++
		} else {
			board.Unattributed++
		}
	}
}

// mcpEpics reads the main agent's open epics from the tasks MCP file.
//
// Returns:
//   - []snapshot.Epic: open epics, the active one first, then the most recently
//     touched
//   - bool: false when the file does not exist
func (p *board) mcpEpics() ([]snapshot.Epic, bool) {
	// No session located, nothing to read
	if p.sessionDir == "" {
		return nil, false
	}
	data, err := os.ReadFile(filepath.Join(p.sessionDir, "tasks.json"))
	// Only a missing file hands over to the built-in list
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false
	}
	var file sessionTasks
	// An unreadable file draws nothing rather than a list from elsewhere
	if err != nil || json.Unmarshal(data, &file) != nil {
		return nil, true
	}
	epics, active := readEpics(file, p.now().Unix())
	known := make(map[int]epicFile, len(epics))
	// The main agent's epics only: subagents keep their own lists; an epic
	// with no agent is the main agent's, as a task with none is
	for _, epic := range epics {
		if epic.Agent == "" || epic.Agent == mainAgent {
			known[epic.ID] = epic
		}
	}
	// An active id naming no epic of the main agent leaves none active
	if _, ok := known[active]; !ok {
		active = snapshot.NoEpic
	}
	byEpic, noEpicTouched := groupTasks(file.Tasks, known)
	open := make([]openEpic, 0, len(known)+1)
	// An epic is open while a task remains, or while it is focused and empty
	for id, epic := range known {
		list := snapshot.TaskList{Items: sortByID(byEpic[id])}
		if list.IsActive() || (id == active && list.Total() == 0) {
			open = append(open, openEpic{
				epic:    snapshot.Epic{ID: id, Title: epic.Title, Active: id == active, Tasks: list},
				touched: epic.Touched,
			})
		}
	}
	// The tasks filed under no epic are one more pill, active when none is
	if list := (snapshot.TaskList{Items: sortByID(byEpic[snapshot.NoEpic])}); list.IsActive() {
		open = append(open, openEpic{
			epic:    snapshot.Epic{ID: snapshot.NoEpic, Active: active == snapshot.NoEpic, Tasks: list},
			touched: noEpicTouched,
		})
	}
	return orderEpics(open), true
}

// readEpics decodes the epics of either file version.
//
// Version 1 kept a map of each agent's current epic and no active map; it is
// read as a list whose epics were all touched now, each agent focused on its
// own.
//
// Params:
//   - file: decoded tasks MCP file
//   - now: Unix time given to epics that carry none
//
// Returns:
//   - []epicFile: every epic of the file
//   - int: the main agent's active epic, NoEpic when none
func readEpics(file sessionTasks, now int64) ([]epicFile, int) {
	var list []epicFile
	// Version 2: a list, focus in its own map
	if json.Unmarshal(file.Epics, &list) == nil {
		return list, file.Active[mainAgent]
	}
	var current map[string]struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
	}
	// Neither shape: no epic at all
	if json.Unmarshal(file.Epics, &current) != nil {
		return nil, file.Active[mainAgent]
	}
	list = make([]epicFile, 0, len(current))
	// Version 1: each agent's current epic is the one it is focused on
	for agent, epic := range current {
		list = append(list, epicFile{ID: epic.ID, Agent: agent, Title: epic.Title, Touched: now})
	}
	return list, current[mainAgent].ID
}

// groupTasks sorts the main agent's tasks into their epics.
//
// Params:
//   - tasks: every task of the file
//   - known: the main agent's epics by id
//
// Returns:
//   - map[int][]snapshot.TaskItem: tasks by epic id; a task naming an epic not
//     on file is left out
//   - int64: last time a task filed under no epic changed
func groupTasks(tasks []taskFile, known map[int]epicFile) (map[int][]snapshot.TaskItem, int64) {
	byEpic := make(map[int][]snapshot.TaskItem)
	var touched int64
	// Subagents keep their own lists; only the main agent's are shown
	for _, task := range tasks {
		if task.Agent != "" && task.Agent != mainAgent {
			continue
		}
		// A task of an epic no longer on file belongs to a finished subject:
		// a version 1 file kept only each agent's current epic
		if _, ok := known[task.Epic]; !ok && task.Epic != snapshot.NoEpic {
			continue
		}
		// The tasks under no epic are ordered by their own last change
		if task.Epic == snapshot.NoEpic {
			touched = max(touched, task.Created, task.Updated)
		}
		byEpic[task.Epic] = append(byEpic[task.Epic], snapshot.TaskItem{ID: task.ID, Subject: task.Subject, Status: task.Status})
	}
	return byEpic, touched
}

// orderEpics puts the active epic first, then the others by last change.
//
// Params:
//   - open: open epics, in any order
//
// Returns:
//   - []snapshot.Epic: the epics in display order
func orderEpics(open []openEpic) []snapshot.Epic {
	// Focus first, then recency, then the newer epic on a tie
	sort.SliceStable(open, func(i, j int) bool {
		a, b := open[i], open[j]
		if a.epic.Active != b.epic.Active {
			return a.epic.Active
		}
		if a.touched != b.touched {
			return a.touched > b.touched
		}
		return a.epic.ID > b.epic.ID
	})
	epics := make([]snapshot.Epic, 0, len(open))
	// Keep only the view the renderer needs
	for _, item := range open {
		epics = append(epics, item.epic)
	}
	return epics
}

// builtinTasks reads the built-in tools' one-file-per-task list.
//
// Returns:
//   - snapshot.TaskList: current list, empty when there is none
func (p *board) builtinTasks() snapshot.TaskList {
	// No list located, nothing to read
	if p.builtinDir == "" {
		return snapshot.TaskList{}
	}
	paths, err := filepath.Glob(filepath.Join(p.builtinDir, "*.json"))
	// An unreadable directory is an empty list
	if err != nil {
		return snapshot.TaskList{}
	}
	items := make([]snapshot.TaskItem, 0, len(paths))
	// Read each task, skipping any that is mid-write or malformed
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var task taskFile
		if err := json.Unmarshal(data, &task); err != nil || task.ID == "" {
			continue
		}
		items = append(items, snapshot.TaskItem{ID: task.ID, Subject: task.Subject, Status: task.Status})
	}
	return snapshot.TaskList{Items: sortByID(items)}
}

// sortByID orders tasks by their numeric id, falling back to text order.
//
// Params:
//   - items: tasks to order in place
//
// Returns:
//   - []snapshot.TaskItem: the same slice, ordered
func sortByID(items []snapshot.TaskItem) []snapshot.TaskItem {
	// Ids are sequence numbers: order them numerically, not as text
	sort.SliceStable(items, func(i, j int) bool {
		a, errA := strconv.Atoi(items[i].ID)
		b, errB := strconv.Atoi(items[j].ID)
		if errA != nil || errB != nil {
			return items[i].ID < items[j].ID
		}
		return a < b
	})
	return items
}

// board reads one session at one instant.
func (s *Store) board(sessionID, listID string, now time.Time) snapshot.TaskBoard {
	return newBoard(sessionID, listID, now).board()
}
