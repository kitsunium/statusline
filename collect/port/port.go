// Package port holds the outbound ports of the collect domain.
//
// Interfaces of design/domains/collect.yaml. This file stands in for the one
// kit generates (port_gen.go) until `kit gen` is available.
package port

import (
	"context"
	"time"

	"github.com/kitsunium/statusline/collect/state"
	"github.com/kitsunium/statusline/ipc"
	"github.com/kitsunium/statusline/quota"
	"github.com/kitsunium/statusline/snapshot"
)

// Repository is collect/port/repository@v1.
type Repository interface {
	Status(dir string) snapshot.GitStatus
	DiffStats(dir string) snapshot.Changes
}

// MCPConfig is collect/port/mcp-config@v1.
type MCPConfig interface {
	Servers(projectDir string, hostPID int) snapshot.MCPServers
}

// MCPCalls is collect/port/mcp-calls@v1.
type MCPCalls interface {
	Busy(transcriptPath, sessionID string, now time.Time) []string
}

// WorkDir is collect/port/workdir@v1.
type WorkDir interface {
	Dir(transcriptPath, fallback string) string
}

// SessionRegistry is collect/port/session-registry@v1.
type SessionRegistry interface {
	Lookup(sessionID string) (state.HostSession, bool)
}

// TaskStore is collect/port/task-store@v1.
type TaskStore interface {
	Board(sessionID, listID string, now time.Time) snapshot.TaskBoard
}

// SystemInfo is collect/port/system-info@v1.
type SystemInfo interface {
	Info() snapshot.System
}

// UsageAPI is collect/port/usage-api@v1.
type UsageAPI interface {
	Fetch(ctx context.Context, token string) (quota.Set, error)
}

// TokenStore is collect/port/token-store@v1.
type TokenStore interface {
	Token(ctx context.Context) (string, error)
}

// StatusPage is collect/port/status-page@v1.
type StatusPage interface {
	Fetch(ctx context.Context) (snapshot.Health, error)
}

// NetworkStore is collect/port/network-store@v1.
type NetworkStore interface {
	LoadNetwork() (state.Network, error)
	SaveNetwork(n state.Network) error
}

// SnapshotCache is collect/port/snapshot-cache@v1.
type SnapshotCache interface {
	SaveSnapshot(key ipc.Key, snap snapshot.Snapshot) error
}

// UpdateStore is collect/port/update-store@v1.
type UpdateStore interface {
	LoadUpdate() (state.Update, error)
	SaveUpdate(u state.Update) error
}

// ReleaseSource is collect/port/release-source@v1.
type ReleaseSource interface {
	Latest(ctx context.Context) (state.Release, error)
	Install(ctx context.Context, rel state.Release) error
	Probe(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// Clock is collect/port/clock@v1.
type Clock interface {
	Now() time.Time
}
