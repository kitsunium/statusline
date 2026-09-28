// Package ipc is the contract between the client and the daemon roles of
// the binary (statusline.ipc/v1): the only way the two processes talk.
//
// Exported API of design/domains/ipc.yaml. This file stands in for the
// shells kit generates (api_gen.go) until `kit gen` is available: every
// exported symbol here only delegates to its unexported twin.
package ipc

import (
	"errors"
	"io"
	"time"

	"github.com/kitsunium/statusline/snapshot"
)

// Protocol names this contract and its major version.
const Protocol string = "statusline.ipc/v1"

// MaxFrame bounds one frame, in bytes.
const MaxFrame int = 1 << 20

// Operations of the contract.
const (
	OpSnapshot string = "snapshot"
	OpStatus   string = "status"
	OpStop     string = "stop"
)

// ErrFrameTooLarge refuses a frame over MaxFrame.
var ErrFrameTooLarge = errors.New("statusline_ipc_frame_too_large: frame over 1 MiB")

// Hello opens a connection, in both directions.
type Hello struct {
	Protocol string `json:"protocol"`
	Version  string `json:"version"`
	PID      int    `json:"pid"`
}

// Key is what a session's state is kept by: session, transcript and
// directory, plus the task list the client's environment names.
type Key struct {
	SessionID      string `json:"session_id,omitempty"`
	TranscriptPath string `json:"transcript_path,omitempty"`
	SessionDir     string `json:"session_dir,omitempty"`
	TaskListID     string `json:"task_list_id,omitempty"`
}

// Hash is a short stable digest of the key, safe in a file name.
func (k Key) Hash() string { return k.hash() }

// Request is one operation.
type Request struct {
	Op  string `json:"op"`
	Key Key    `json:"key"`
}

// Network is the daemon's network bookkeeping, as status reports it.
type Network struct {
	UsageFetchedAt  time.Time `json:"usage_fetched_at,omitzero"`
	UsageAttemptAt  time.Time `json:"usage_attempt_at,omitzero"`
	UsageRetryAfter time.Time `json:"usage_retry_after,omitzero"`
	UsageStatus     int       `json:"usage_status,omitempty"`
	HealthFetchedAt time.Time `json:"health_fetched_at,omitzero"`
	HealthAttemptAt time.Time `json:"health_attempt_at,omitzero"`
}

// Status describes a running daemon.
type Status struct {
	Version     string    `json:"version"`
	PID         int       `json:"pid"`
	Executable  string    `json:"executable"`
	StartedAt   time.Time `json:"started_at"`
	LastRequest time.Time `json:"last_request,omitzero"`
	Sessions    int       `json:"sessions"`
	Network     Network   `json:"network"`
	BadVersion  string    `json:"bad_version,omitempty"`
}

// Response answers a Request.
type Response struct {
	Snapshot *snapshot.Snapshot `json:"snapshot,omitempty"`
	Status   *Status            `json:"status,omitempty"`
	Error    string             `json:"error,omitempty"`
}

// Instance is where one daemon instance lives.
type Instance struct {
	Dir    string
	Socket string
	Lock   string
	Cache  string
	State  string
	Log    string
	// PID names the running daemon's process.
	PID string
	// Heartbeat is touched by the daemon on every tick: a daemon that holds
	// the socket but lets it go stale is stuck, and the client replaces it.
	Heartbeat string
}

// CachePath is the file holding the last snapshot of a key.
func (i Instance) CachePath(key Key) string { return i.cachePath(key) }

// LocateInput identifies an instance: one per (UID, host configuration
// directory, executable path).
type LocateInput struct {
	RuntimeDir string
	UID        int
	ConfigDir  string
	Executable string
}

// Locate derives an instance's paths; both roles compute the same ones.
func Locate(in LocateInput) Instance { return locate(in) }

// WriteFrame writes one length-prefixed JSON frame.
func WriteFrame(w io.Writer, v any) error { return writeFrame(w, v) }

// ReadFrame reads one length-prefixed JSON frame.
func ReadFrame(r io.Reader, v any) error { return readFrame(r, v) }

// CompareVersions orders two versions: -1, 0 or 1. A development build
// (empty or "dev") is older than any release.
func CompareVersions(a, b string) int { return compareVersions(a, b) }

// Compatible reports whether a peer's protocol has this contract's major.
func Compatible(protocol string) bool { return compatible(protocol) }

// Here locates the instance of the running executable: the host
// configuration directory is CLAUDE_CONFIG_DIR, else ~/.claude; the runtime
// directory is XDG_RUNTIME_DIR, else the temporary directory.
func Here(getenv func(string) string) (Instance, error) { return here(getenv) }
