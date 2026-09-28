// Package transcripts reads the ends of the session transcripts: which MCP
// servers are being called, and where the session is actually working.
//
// Exported API of design/domains/collect.yaml (collect/component/transcripts).
// This file stands in for the shells kit generates (api_gen.go) until
// `kit gen` is available: every exported symbol only delegates to its
// unexported twin.
package transcripts

import "time"

// Reader implements collect/port/mcp-calls@v1 and collect/port/workdir@v1.
type Reader struct{}

// New returns a transcript reader.
func New() *Reader { return &Reader{} }

// Busy returns the tool-name keys of the MCP servers with a call in flight,
// or one that returned less than 2 s ago, in the main transcript and those
// of the running subagents.
func (r *Reader) Busy(transcriptPath, sessionID string, now time.Time) []string {
	return r.busy(transcriptPath, sessionID, now)
}

// Dir returns the last existing location a tool call named, climbed to its
// git root; fallback when none.
func (r *Reader) Dir(transcriptPath, fallback string) string { return r.dir(transcriptPath, fallback) }
