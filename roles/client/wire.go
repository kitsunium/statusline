// Package client wires the client role (D22): every adapter bound to the
// port it implements, as design/product.yaml's bindings say.
//
// This file stands in for the wiring kit generates (wire_gen.go) until
// `kit gen` is available.
package client

import (
	"os"
	"path/filepath"

	"github.com/kitsunium/statusline/ipc"
	"github.com/kitsunium/statusline/render/cli"
	"github.com/kitsunium/statusline/render/ctl"
	"github.com/kitsunium/statusline/render/daemonlink"
	"github.com/kitsunium/statusline/render/show"
	"github.com/kitsunium/statusline/render/systemclock"
)

// Build is what the binary knows about itself.
type Build struct {
	Version string
}

// Main runs the client role and returns its exit code.
func Main(b Build) int {
	getenv := os.Getenv
	// Without an instance the line still renders, from stdin alone
	instance, _ := ipc.Here(getenv)
	link := daemonlink.New(daemonlink.Config{Instance: instance, Version: b.Version, Executable: executable()})
	clock := systemclock.New()
	entry := cli.New(cli.Deps{
		Show:   show.New(show.Deps{Clock: clock, Snapshots: link}),
		Status: ctl.NewStatus(ctl.Deps{Daemon: link}),
		Stop:   ctl.NewStop(ctl.Deps{Daemon: link}),
	})
	return entry.Run(cli.RunInput{Args: os.Args, Stdin: os.Stdin, Stdout: os.Stdout, Getenv: getenv, Version: b.Version})
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
