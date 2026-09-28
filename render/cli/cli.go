package cli

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/kitsunium/statusline/render/ctl"
	"github.com/kitsunium/statusline/render/show"
)

// devVersion names a build without a version.
const devVersion string = "dev"

// run: -v/--version, `daemon status`, `daemon stop`; anything else renders,
// as the legacy binary did for arguments it did not know.
func (c *CLI) run(in RunInput) int {
	ctx := context.Background()
	args := in.Args
	if len(args) > 0 {
		args = args[1:]
	}
	switch {
	case len(args) == 1 && (args[0] == "-v" || args[0] == "--version"):
		version := in.Version
		if version == "" {
			version = devVersion
		}
		_, _ = fmt.Fprintln(in.Stdout, "statusline", version)
		return 0
	case len(args) == 2 && args[0] == "daemon" && args[1] == "status":
		return c.status(ctx, in.Stdout)
	case len(args) == 2 && args[0] == "daemon" && args[1] == "stop":
		out, _ := c.deps.Stop.Execute(ctx, ctl.StopInput{})
		if out.WasRunning {
			_, _ = fmt.Fprintln(in.Stdout, "daemon stopped")
		} else {
			_, _ = fmt.Fprintln(in.Stdout, "no daemon running")
		}
		return 0
	}
	// An unreadable stdin still renders: the zero payload is a status line
	data, _ := io.ReadAll(in.Stdin)
	out, _ := c.deps.Show.Execute(ctx, show.ShowInput{
		Stdin:      data,
		Columns:    in.Getenv("COLUMNS"),
		TaskListID: in.Getenv("CLAUDE_CODE_TASK_LIST_ID"),
		Getenv:     in.Getenv,
	})
	_, _ = io.WriteString(in.Stdout, out.Line)
	return 0
}

// status prints the daemon's state; exit 1 when none runs, for scripts.
func (c *CLI) status(ctx context.Context, w io.Writer) int {
	out, _ := c.deps.Status.Execute(ctx, ctl.StatusInput{})
	if !out.Running {
		_, _ = fmt.Fprintln(w, "no daemon running")
		return 1
	}
	st := out.Status
	_, _ = fmt.Fprintf(w, "daemon %s pid %d, up %s, %d session(s)\n", st.Version, st.PID, time.Since(st.StartedAt).Round(time.Second), st.Sessions)
	_, _ = fmt.Fprintf(w, "executable %s\n", st.Executable)
	n := st.Network
	_, _ = fmt.Fprintf(w, "usage: fetched %s, attempted %s, status %d, retry after %s\n", stamp(n.UsageFetchedAt), stamp(n.UsageAttemptAt), n.UsageStatus, stamp(n.UsageRetryAfter))
	_, _ = fmt.Fprintf(w, "health: fetched %s, attempted %s\n", stamp(n.HealthFetchedAt), stamp(n.HealthAttemptAt))
	if st.BadVersion != "" {
		_, _ = fmt.Fprintf(w, "bad version (never installed again): %s\n", st.BadVersion)
	}
	return 0
}

// stamp prints an instant, or "never".
func stamp(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	return t.Local().Format(time.RFC3339)
}
