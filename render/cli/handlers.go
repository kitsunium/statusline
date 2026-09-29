// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/kitsunium/sdk/framework/kit"

	"github.com/kitsunium/statusline/render/show"
)

// devVersion names a build without a version.
const devVersion string = "dev"

// line is the client's only command, and its default: -v/--version prints
// the version; anything else renders, as the legacy binary did for
// arguments it did not know. It never fails: this process is the host's
// prompt line, and a render that errs still prints what it could.
func line(ctx context.Context, args []string, std kit.Stdio) int {
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
