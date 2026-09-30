// Package mcpcalls reads which MCP servers the session is calling right now.
package transcripts

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// Scanning constants.
const (
	// configDirEnv relocates the whole configuration directory.
	configDirEnv string = "CLAUDE_CONFIG_DIR"
	// tailSize is how much of each transcript end is read: a call in flight
	// is always among the last lines.
	callsTailSize int64 = 128 << 10
	// linger keeps a server lit after its call returned. The line redraws
	// once per second: without it a short call would never show.
	linger time.Duration = 2 * time.Second
	// staleCall is how long a call may wait for its result before it is
	// taken for one whose result was lost (a crash, a killed session).
	staleCall time.Duration = 30 * time.Minute
	// staleAgent matches the tasks adapter: a subagent without a stop event
	// for that long is taken for one whose stop was lost.
	staleAgent time.Duration = 12 * time.Hour
	// toolPrefix opens every MCP tool name: mcp__<server>__<tool>.
	toolPrefix string = "mcp__"
)

// Byte markers used to skip lines without decoding them.
var (
	mcpMarker    = []byte(`"mcp__`)
	resultMarker = []byte(`"tool_result"`)
)

// unsafeID matches the characters replaced in a session directory name, and
// agentID the only agent ids turned into a path. Compiled on first use: the
// client links this package but never runs it.
var (
	unsafeID = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`[^a-zA-Z0-9_-]`) })
	agentID  = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`^[a-zA-Z0-9_-]+$`) })
)

// calls scans one session's transcripts for MCP calls.
//
// The main transcript is the one stdin names; each running subagent (from
// the kodflow-hooks agents.json) writes its own under
// <transcript dir>/<session>/subagents/agent-<id>.jsonl. Only their tails are
// read; nothing is executed and no hook is involved.
type calls struct {
	transcriptPath string
	subagentDir    string
	agentsFile     string
	now            func() time.Time
}

// line is the part of a transcript record this package reads.
type callLine struct {
	Timestamp string `json:"timestamp"`
	Message   struct {
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

// block is one content block: a tool call or a tool result.
type callBlock struct {
	Type      string `json:"type"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	ToolUseID string `json:"tool_use_id"`
}

// sessionAgents is the kodflow-hooks running-agents file.
type sessionAgents struct {
	Agents map[string]struct {
		Started int64  `json:"started"`
		Stopped *int64 `json:"stopped"`
	} `json:"agents"`
}

// call is a tool call awaiting its result.
type call struct {
	server string
	at     time.Time
}

// newCalls locates the transcripts of a session.
//
// Params:
//   - transcriptPath: main transcript from stdin, may be empty
//   - sessionID: session identifier from stdin, may be empty
//
// Returns:
//   - *calls: scanner reading nothing without a transcript
func newCalls(transcriptPath, sessionID string, now time.Time) *calls {
	p := &calls{transcriptPath: transcriptPath, now: func() time.Time { return now }}
	// Subagents need both the session and where its transcripts live
	if transcriptPath == "" || sessionID == "" {
		return p
	}
	p.subagentDir = filepath.Join(filepath.Dir(transcriptPath), sessionID, "subagents")
	base := os.Getenv(configDirEnv)
	// Default to the per-user configuration directory
	if base == "" {
		home, err := os.UserHomeDir()
		// Without a home there is no agents registry to read
		if err != nil {
			return p
		}
		base = filepath.Join(home, ".claude")
	}
	p.agentsFile = filepath.Join(base, "kodflow", "sessions", unsafeID().ReplaceAllString(sessionID, "-"), "agents.json")
	return p
}

// Busy returns the servers with a call in flight or one that just ended.
//
// Returns:
//   - []string: distinct tool-name keys, sorted
func (p *calls) busy() []string {
	// Without a transcript nothing can be seen
	if p.transcriptPath == "" {
		return nil
	}
	now := p.now()
	busy := make(map[string]bool)
	// The main agent and every running subagent
	for _, path := range append([]string{p.transcriptPath}, p.subagentTranscripts(now)...) {
		scanFile(path, now, busy)
	}
	keys := make([]string, 0, len(busy))
	// Collect the set in a stable order
	for key := range busy {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// subagentTranscripts lists the transcripts of the running subagents.
//
// Params:
//   - now: current time
//
// Returns:
//   - []string: transcript paths, possibly of files not written yet
func (p *calls) subagentTranscripts(now time.Time) []string {
	// No registry located, no subagent to follow
	if p.agentsFile == "" {
		return nil
	}
	data, err := os.ReadFile(p.agentsFile)
	// No registry yet means no subagent has started
	if err != nil {
		return nil
	}
	var reg sessionAgents
	// A registry mid-write reads as empty until the next redraw
	if json.Unmarshal(data, &reg) != nil {
		return nil
	}
	cutoff := now.Add(-staleAgent).Unix()
	paths := make([]string, 0, len(reg.Agents))
	// Keep the agents still running with an id safe to put in a path
	for id, agent := range reg.Agents {
		// A stopped, stale or oddly named agent is skipped
		if agent.Stopped != nil || agent.Started < cutoff || !agentID().MatchString(id) {
			continue
		}
		paths = append(paths, filepath.Join(p.subagentDir, "agent-"+id+".jsonl"))
	}
	sort.Strings(paths)
	return paths
}

// scanFile adds to busy the servers one transcript is calling.
//
// A call is in flight while its tool_use has no tool_result yet; a returned
// call keeps its server lit for linger after the result's timestamp.
//
// Params:
//   - path: transcript to read
//   - now: current time
//   - busy: set of server keys to fill
func scanFile(path string, now time.Time, busy map[string]bool) {
	tail, err := tail(path, callsTailSize)
	// A transcript not written yet calls nothing
	if err != nil {
		return
	}
	pending := make(map[string]call)
	// Walk forward so a result always meets its call first
	for _, raw := range bytes.Split(tail, []byte("\n")) {
		hasCall := bytes.Contains(raw, mcpMarker)
		// A result only matters when a call is waiting for it
		if !hasCall && (len(pending) == 0 || !bytes.Contains(raw, resultMarker)) {
			continue
		}
		scanLine(raw, now, pending, busy)
	}
	// Calls still waiting are in flight, unless they waited too long
	for _, c := range pending {
		// A zero time (no timestamp) is trusted as recent
		if c.at.IsZero() || now.Sub(c.at) <= staleCall {
			busy[c.server] = true
		}
	}
}

// scanLine records the MCP calls and the results of one record.
//
// Params:
//   - raw: one JSONL record
//   - now: current time
//   - pending: calls awaiting their result, updated
//   - busy: set of server keys, filled with recently returned calls
func scanLine(raw []byte, now time.Time, pending map[string]call, busy map[string]bool) {
	var rec callLine
	// A torn or malformed line says nothing
	if json.Unmarshal(raw, &rec) != nil {
		return
	}
	var blocks []callBlock
	// Plain-text content carries no call and no result
	if json.Unmarshal(rec.Message.Content, &blocks) != nil {
		return
	}
	at, _ := time.Parse(time.RFC3339Nano, rec.Timestamp)
	// Each block is a call, a result, or neither
	for _, b := range blocks {
		// Sort the block by kind
		switch b.Type {
		case "tool_use":
			// Only MCP tools name a server
			if server := serverKey(b.Name); server != "" && b.ID != "" {
				pending[b.ID] = call{server: server, at: at}
			}
		case "tool_result":
			c, waiting := pending[b.ToolUseID]
			// A result whose call is outside the tail is ignored
			if !waiting {
				continue
			}
			delete(pending, b.ToolUseID)
			// A call that just returned still shows for a moment
			if !at.IsZero() && now.Sub(at) <= linger {
				busy[c.server] = true
			}
		}
	}
}

// serverKey extracts the server part of an MCP tool name.
//
// Params:
//   - name: tool name, mcp__<server>__<tool>
//
// Returns:
//   - string: server key, empty for a tool that is not MCP
func serverKey(name string) string {
	rest, ok := strings.CutPrefix(name, toolPrefix)
	// Built-in tools are not MCP calls
	if !ok {
		return ""
	}
	server, _, found := strings.Cut(rest, "__")
	// A name without the tool part is not a well-formed MCP tool
	if !found {
		return ""
	}
	return server
}
