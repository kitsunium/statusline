// Package daemon wires the daemon role (D22): every adapter bound to the
// port it implements, as design/product.yaml's bindings say.
//
// This file stands in for the wiring kit generates (wire_gen.go) until
// `kit gen` is available.
package daemon

import (
	"encoding/base64"
	"os"
	"path/filepath"

	"github.com/kitsunium/statusline/collect/credentials"
	collectdaemon "github.com/kitsunium/statusline/collect/daemon"
	"github.com/kitsunium/statusline/collect/gather"
	"github.com/kitsunium/statusline/collect/gitcli"
	"github.com/kitsunium/statusline/collect/mcpconfig"
	"github.com/kitsunium/statusline/collect/refresh"
	"github.com/kitsunium/statusline/collect/releases"
	"github.com/kitsunium/statusline/collect/sessions"
	"github.com/kitsunium/statusline/collect/state"
	"github.com/kitsunium/statusline/collect/statefile"
	"github.com/kitsunium/statusline/collect/statuspage"
	"github.com/kitsunium/statusline/collect/sysinfo"
	"github.com/kitsunium/statusline/collect/systemclock"
	"github.com/kitsunium/statusline/collect/taskstore"
	"github.com/kitsunium/statusline/collect/transcripts"
	"github.com/kitsunium/statusline/collect/update"
	"github.com/kitsunium/statusline/collect/usageapi"
	"github.com/kitsunium/statusline/ipc"
)

// Build is what the binary knows about itself.
type Build struct {
	Version string
	// VendorKey is the base64 ed25519 key releases are signed with.
	VendorKey string
}

// Main runs the daemon role and returns its exit code.
func Main(b Build) int {
	getenv := os.Getenv
	instance, err := ipc.Here(getenv)
	if err != nil {
		return 1
	}
	exe := executable()
	key, _ := base64.StdEncoding.DecodeString(b.VendorKey)

	clock := systemclock.New()
	store := statefile.New(instance)
	registry := state.NewRegistry()
	reader := transcripts.New()
	refresher := refresh.New(refresh.Deps{
		Clock:  clock,
		Store:  store,
		Tokens: credentials.New(getenv),
		Usage:  usageapi.New(getenv),
		Status: statuspage.New(getenv),
	})
	collector := gather.New(gather.Deps{
		Clock:      clock,
		WorkDir:    reader,
		Sessions:   sessions.New(),
		Repository: gitcli.New(),
		MCPConfig:  mcpconfig.New(),
		MCPCalls:   reader,
		Tasks:      taskstore.New(),
		System:     sysinfo.New(),
		Cache:      store,
		Network:    refresher.Latest(),
		Registry:   registry,
	})
	updater := update.New(update.Deps{
		Clock:    clock,
		Store:    store,
		Releases: releases.New(releases.Config{Version: b.Version, Executable: exe, VendorKey: key}),
	})
	entry := collectdaemon.New(collectdaemon.Deps{
		Clock:    clock,
		Registry: registry,
		Collect:  collector,
		Refresh:  refresher,
		Latest:   refresher.Latest(),
		Update:   updater,
		Updates:  store,
	})
	return entry.Run(collectdaemon.RunInput{Version: b.Version, Executable: exe, Instance: instance, Getenv: getenv})
}

// executable is this binary, symbolic links resolved.
func executable() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		return resolved
	}
	return exe
}
