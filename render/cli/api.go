// Package cli is the client role's entry: argv to use case, stdin in,
// stdout out, and never a non-zero exit on a bad payload — this process is
// the host's prompt line, and a raw error would replace it on every redraw.
//
// Exported API of design/domains/render.yaml (render/component/cli). This
// file stands in for the shells kit generates (api_gen.go) until `kit gen`
// is available: every exported symbol only delegates to its unexported twin.
package cli

import (
	"io"

	"github.com/kitsunium/statusline/render/ctl"
	"github.com/kitsunium/statusline/render/show"
)

// RunInput is one invocation of the client.
type RunInput struct {
	Args    []string
	Stdin   io.Reader
	Stdout  io.Writer
	Getenv  func(string) string
	Version string
}

// Deps are the use cases the commands run.
type Deps struct {
	Show   show.ShowStatusLine
	Status ctl.DaemonStatus
	Stop   ctl.StopDaemon
}

// CLI dispatches the client's commands.
type CLI struct {
	deps Deps
}

// New returns the client's entry.
func New(deps Deps) *CLI { return &CLI{deps: deps} }

// Run executes one invocation and returns the exit code.
func (c *CLI) Run(in RunInput) int { return c.run(in) }
