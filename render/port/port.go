// Package port holds the outbound ports of the render domain.
//
// Interfaces of design/domains/render.yaml. This file stands in for the one
// kit generates (port_gen.go) until `kit gen` is available.
package port

import (
	"context"
	"time"

	"github.com/kitsunium/statusline/ipc"
	"github.com/kitsunium/statusline/snapshot"
)

// SnapshotOrigin says where a snapshot came from.
type SnapshotOrigin string

// Snapshot origins.
const (
	OriginDaemon SnapshotOrigin = "daemon"
	OriginCache  SnapshotOrigin = "cache"
	OriginNone   SnapshotOrigin = "none"
)

// SnapshotSource is render/port/snapshot-source@v1.
type SnapshotSource interface {
	Snapshot(ctx context.Context, key ipc.Key) (snapshot.Snapshot, SnapshotOrigin)
}

// DaemonControl is render/port/daemon-control@v1.
type DaemonControl interface {
	Status(ctx context.Context) (ipc.Status, error)
	Stop(ctx context.Context) error
}

// Clock is render/port/clock@v1.
type Clock interface {
	Now() time.Time
}
