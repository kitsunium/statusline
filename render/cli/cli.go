package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/kitsunium/statusline/render/show"
)

// devVersion names a build without a version.
const devVersion string = "dev"

// cli is the CLI's own state: its use cases, and the process's streams and
// environment, which tests replace.
type cli struct {
	show    show.ShowStatusLine
	version string
	stdin   io.Reader
	stdout  io.Writer
	environ func() []string
}

func newCLI(showStatusLine show.ShowStatusLine, version string) *CLI {
	return &CLI{cli{show: showStatusLine, version: version, stdin: os.Stdin, stdout: os.Stdout, environ: os.Environ}}
}

// run: -v/--version; anything else renders, as the legacy binary did for
// arguments it did not know (`daemon …` is the daemon role's). A render
// never fails: this process is the host's prompt line.
func (a *CLI) run(ctx context.Context, args []string) error {
	switch {
	case len(args) == 1 && (args[0] == "-v" || args[0] == "--version"):
		version := a.version
		if version == "" {
			version = devVersion
		}
		_, _ = fmt.Fprintln(a.stdout, "statusline", version)
		return nil
	}
	// An unreadable stdin still renders: the zero payload is a status line
	data, _ := io.ReadAll(a.stdin)
	out, _ := a.show.Execute(ctx, show.ShowStatusLineInput{Stdin: data, Env: environment(a.environ())})
	_, _ = io.WriteString(a.stdout, out.Line)
	return nil
}

// environment turns KEY=VALUE pairs into a map; the last one wins.
func environment(pairs []string) map[string]string {
	env := make(map[string]string, len(pairs))
	for _, kv := range pairs {
		if k, v, ok := strings.Cut(kv, "="); ok {
			env[k] = v
		}
	}
	return env
}
