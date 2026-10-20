// Written once from the design; this file is yours. The shells in
// gen_design.go delegate to what it holds; a body that still answers
// errNotImplemented is a stub.

package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/kitsunium/sdk/framework/app"

	"github.com/kitsunium/statusline/render/show"
)

// devVersion names a build without a version.
const devVersion string = "dev"

// line is the client's only command, and its default: -v/--version prints
// the version; anything else renders, as the legacy binary did for
// arguments it did not know. It never fails: this process is the host's
// prompt line, and a render that errs still prints what it could.
func line(ctx context.Context, args []string, std app.Stdio) int {
	if len(args) == 1 && (args[0] == "-v" || args[0] == "--version") {
		version := wired.version
		if version == "" {
			version = devVersion
		}
		_, _ = fmt.Fprintln(std.Out, "statusline", version)
		return 0
	}
	// An unreadable stdin still renders: the zero payload is a status line
	data, _ := io.ReadAll(std.In)
	out, _ := wired.showStatusLine.Execute(ctx, show.ShowStatusLineInput{Stdin: data, Env: environment(os.Environ())})
	_, _ = io.WriteString(std.Out, out.Line)
	return 0
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
