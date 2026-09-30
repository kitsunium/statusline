package transcripts

import (
	"context"
	"time"
)

// transcripts holds nothing: every call reads the tails afresh.
type transcripts struct{}

func newTranscripts() *Transcripts { return &Transcripts{} }

// busy scans one session at one instant.
func (a *Transcripts) busy(_ context.Context, transcriptPath, sessionID string, now time.Time) ([]string, error) {
	return newCalls(transcriptPath, sessionID, now).busy(), nil
}

// dir is where the session works, the fallback when no tool call says.
func (a *Transcripts) dir(_ context.Context, transcriptPath, fallback string) (string, error) {
	return dir(transcriptPath, fallback), nil
}
