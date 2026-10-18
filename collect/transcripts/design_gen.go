// Code generated from design/v1; DO NOT EDIT.
// design:digest sha256:f7116793c8d8ac76611661ce6107f6662b6eb477309df7bb89556666304dc4e6

// Package transcripts — The ends of the transcripts: the MCP calls in
// flight, and where the session works.
//
// Its structure is generated from design/ (ADR 0010); its bodies are the
// hand-written twins beside this file.
package transcripts

import (
	"context"
	"time"

	"github.com/kitsunium/sdk/pkg/v1/errs"
	"github.com/kitsunium/statusline/collect/collector"
)

// errNotImplemented is what a skeleton answers until its body is written:
// a twin that still returns it is a stub.
var errNotImplemented = errs.New(errs.Pack(64, 3, 6, 255), "NOT_IMPLEMENTED", "not implemented yet", "transcripts: a hand-written body is still a skeleton")

// Transcripts is an adapter of the design.
type Transcripts struct {
	transcripts // its own state, hand-written
}

// NewTranscripts builds Transcripts: its body is the hand-written newTranscripts.
func NewTranscripts() *Transcripts {
	return newTranscripts()
}

// Busy is collect/port/MCPCalls@v1's.
func (a *Transcripts) Busy(ctx context.Context, transcriptPath string, sessionID string, now time.Time) ([]string, error) {
	return a.busy(ctx, transcriptPath, sessionID, now)
}

var _ collector.MCPCallsV1 = (*Transcripts)(nil)

// Dir is collect/port/WorkDir@v1's.
func (a *Transcripts) Dir(ctx context.Context, transcriptPath string, fallback string) (string, error) {
	return a.dir(ctx, transcriptPath, fallback)
}

var _ collector.WorkDirV1 = (*Transcripts)(nil)
