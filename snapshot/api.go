// Package snapshot holds what the daemon collects for one session and the
// client renders: git, MCP servers, tasks, health, system, update notice.
//
// Exported API of design/domains/snapshot.yaml. This file stands in for the
// shells kit generates (api_gen.go) until `kit gen` is available: every
// exported symbol here only delegates to its unexported twin.
package snapshot

import (
	"time"

	"github.com/kitsunium/statusline/quota"
)

// GitStatus is the state of the repository the session works in.
type GitStatus struct {
	Branch    string `json:"branch,omitempty"`
	Modified  int    `json:"modified,omitempty"`
	Untracked int    `json:"untracked,omitempty"`
	Worktrees int    `json:"worktrees,omitempty"`
}

// IsInRepo reports whether a branch was found.
func (s GitStatus) IsInRepo() bool { return s.isInRepo() }

// Changes counts lines added and removed against HEAD.
type Changes struct {
	Added   int `json:"added,omitempty"`
	Removed int `json:"removed,omitempty"`
}

// HasChanges reports whether any line changed.
func (c Changes) HasChanges() bool { return c.hasChanges() }

// HasAdded reports whether lines were added.
func (c Changes) HasAdded() bool { return c.hasAdded() }

// HasRemoved reports whether lines were removed.
func (c Changes) HasRemoved() bool { return c.hasRemoved() }

// MCPSource names the configuration scope that declared an MCP server.
type MCPSource string

// Configuration scopes, strongest first.
const (
	MCPSourceUnknown MCPSource = ""
	MCPSourceManaged MCPSource = "managed"
	MCPSourceCLI     MCPSource = "cli"
	MCPSourceLocal   MCPSource = "local"
	MCPSourceProject MCPSource = "project"
	MCPSourceUser    MCPSource = "user"
	MCPSourcePlugin  MCPSource = "plugin"
)

// MCPServer is one MCP server the session can reach.
type MCPServer struct {
	Name    string    `json:"name"`
	Enabled bool      `json:"enabled,omitempty"`
	Plugin  string    `json:"plugin,omitempty"`
	Source  MCPSource `json:"source,omitempty"`
	Busy    bool      `json:"busy,omitempty"`
}

// MCPServers is a list of MCP servers, in precedence order.
type MCPServers []MCPServer

// WithSource tags every server with its scope, in place.
func (s MCPServers) WithSource(src MCPSource) MCPServers { return s.withSource(src) }

// WithBusy returns a copy with the servers being called lit, unknown ones
// appended.
func (s MCPServers) WithBusy(keys []string) MCPServers { return s.withBusy(keys) }

// ToolKey normalises a server or plugin name the way tool names spell it.
func ToolKey(name string) string { return toolKey(name) }

// Task statuses as written by the task tools.
const (
	TaskPending    string = "pending"
	TaskInProgress string = "in_progress"
	TaskWaiting    string = "waiting"
	TaskCompleted  string = "completed"
)

// TaskItem is one task.
type TaskItem struct {
	ID      string `json:"id"`
	Subject string `json:"subject,omitempty"`
	Status  string `json:"status,omitempty"`
}

// TaskList is a list of tasks in creation order.
type TaskList struct {
	Items []TaskItem `json:"items,omitempty"`
}

// Total returns how many tasks the list holds.
func (l TaskList) Total() int { return l.total() }

// Done returns how many tasks are completed.
func (l TaskList) Done() int { return l.done() }

// Active returns how many tasks are in progress.
func (l TaskList) Active() int { return l.active() }

// Waiting returns how many tasks are blocked on the user.
func (l TaskList) Waiting() int { return l.waiting() }

// Headline returns the task to show and its status.
func (l TaskList) Headline() (string, string) { return l.headline() }

// Current returns the first in-progress subject.
func (l TaskList) Current() string { return l.current() }

// IsActive reports whether a task remains open.
func (l TaskList) IsActive() bool { return l.isActive() }

// NoEpic is the epic of tasks tied to none.
const NoEpic int = 0

// Epic is an open epic of the main agent.
type Epic struct {
	ID        int      `json:"id"`
	Title     string   `json:"title,omitempty"`
	Active    bool     `json:"active,omitempty"`
	Tasks     TaskList `json:"tasks"`
	Subagents int      `json:"subagents,omitempty"`
}

// TaskBoard is the session's open epics and running subagents.
type TaskBoard struct {
	Epics        []Epic `json:"epics,omitempty"`
	Unattributed int    `json:"unattributed,omitempty"`
}

// IsEmpty reports whether nothing is open and no subagent runs.
func (b TaskBoard) IsEmpty() bool { return b.isEmpty() }

// Health is the aggregate state of the provider's public services.
type Health int

// Service health levels.
const (
	HealthUnknown Health = iota
	HealthOK
	HealthDegraded
	HealthDown
)

// ClassifyHealth aggregates component states into a level.
func ClassifyHealth(states []string) Health { return classifyHealth(states) }

// OS is the operating system family.
type OS int

// Operating systems.
const (
	OSLinux OS = iota
	OSDarwin
	OSWindows
	OSUnknown
)

// System is the operating system and whether it runs in a container.
type System struct {
	OS       OS   `json:"os"`
	IsDocker bool `json:"docker,omitempty"`
}

// UpdateNotice says an update is being installed.
type UpdateNotice struct {
	Available bool   `json:"available,omitempty"`
	Version   string `json:"version,omitempty"`
}

// Snapshot is everything collected for one session key.
type Snapshot struct {
	WorkDir     string       `json:"work_dir,omitempty"`
	Git         GitStatus    `json:"git"`
	Changes     Changes      `json:"changes"`
	MCP         MCPServers   `json:"mcp,omitempty"`
	Tasks       TaskBoard    `json:"tasks"`
	Working     bool         `json:"working,omitempty"`
	System      System       `json:"system"`
	Health      Health       `json:"health,omitempty"`
	API         quota.Set    `json:"api"`
	Update      UpdateNotice `json:"update"`
	CollectedAt time.Time    `json:"collected_at"`
}
