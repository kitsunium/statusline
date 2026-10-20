// Code generated from design/v1; DO NOT EDIT.
// design:digest sha256:f7116793c8d8ac76611661ce6107f6662b6eb477309df7bb89556666304dc4e6

// Package mcpconfig — The MCP servers from the managed file, the host's
// command line, the local, project and user scopes and the enabled plugins.
//
// Its structure is generated from design/ (ADR 0010); its bodies are the
// hand-written twins beside this file.
package mcpconfig

import (
	"context"

	"github.com/kitsunium/sdk/pkg/v1/errs"
	"github.com/kitsunium/statusline/collect/collector"
	"github.com/kitsunium/statusline/snapshot"
)

// errNotImplemented is what a skeleton answers until its body is written:
// a twin that still returns it is a stub.
var errNotImplemented = errs.New(errs.Pack(64, 3, 5, 255), "NOT_IMPLEMENTED", "not implemented yet", "mcpconfig: a hand-written body is still a skeleton")

// Config is an adapter of the design.
type Config struct {
	config // its own state, hand-written
}

// NewConfig builds Config: its body is the hand-written newConfig.
func NewConfig() *Config {
	return newConfig()
}

// Servers is collect/port/MCPConfig@v1's.
func (a *Config) Servers(ctx context.Context, projectDir string, hostPID int) (snapshot.MCPServers, error) {
	return a.servers(ctx, projectDir, hostPID)
}

var _ collector.MCPConfigV1 = (*Config)(nil)
