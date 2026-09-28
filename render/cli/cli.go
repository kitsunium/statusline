package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/kitsunium/statusline/render/show"
)

// devVersion names a build without a version.
const devVersion string = "dev"

// cli is the CLI's own state: its use cases, and the process's streams and
// environment, which tests replace.
type cli struct {
	show    show.ShowStatusLine
	status  show.DaemonStatus
	stop    show.StopDaemon
	stdin   io.Reader
	stdout  io.Writer
	environ func() []string
	version func() string
}

func newCLI(showStatusLine show.ShowStatusLine, daemonStatus show.DaemonStatus, stopDaemon show.StopDaemon) *CLI {
	return &CLI{cli{
		show: showStatusLine, status: daemonStatus, stop: stopDaemon,
		stdin: os.Stdin, stdout: os.Stdout, environ: os.Environ, version: buildVersion,
	}}
}

// run: -v/--version, `daemon status`, `daemon stop`; anything else renders,
// as the legacy binary did for arguments it did not know. It returns an
// error only for `daemon status` without a daemon, for scripts: a render
// never fails, this process is the host's prompt line.
func (a *CLI) run(ctx context.Context, args []string) error {
	switch {
	case len(args) == 1 && (args[0] == "-v" || args[0] == "--version"):
		version := a.version()
		if version == "" {
			version = devVersion
		}
		_, _ = fmt.Fprintln(a.stdout, "statusline", version)
		return nil
	case len(args) == 2 && args[0] == "daemon" && args[1] == "status":
		return a.daemonStatus(ctx)
	case len(args) == 2 && args[0] == "daemon" && args[1] == "stop":
		out, _ := a.stop.Execute(ctx, show.StopDaemonInput{})
		if out.WasRunning {
			_, _ = fmt.Fprintln(a.stdout, "daemon stopped")
		} else {
			_, _ = fmt.Fprintln(a.stdout, "no daemon running")
		}
		return nil
	}
	// An unreadable stdin still renders: the zero payload is a status line
	data, _ := io.ReadAll(a.stdin)
	out, _ := a.show.Execute(ctx, show.ShowStatusLineInput{Stdin: data, Env: environment(a.environ())})
	_, _ = io.WriteString(a.stdout, out.Line)
	return nil
}

// errNoDaemon makes `daemon status` exit non-zero when none runs.
var errNoDaemon = fmt.Errorf("no daemon running")

// daemonStatus prints the daemon's state.
func (a *CLI) daemonStatus(ctx context.Context) error {
	out, _ := a.status.Execute(ctx, show.DaemonStatusInput{})
	if !out.Running {
		_, _ = fmt.Fprintln(a.stdout, "no daemon running")
		return errNoDaemon
	}
	st := out.Status
	_, _ = fmt.Fprintf(a.stdout, "daemon %s pid %d, up %s, %d session(s)\n", st.Version, st.PID, time.Since(st.StartedAt).Round(time.Second), st.Sessions)
	_, _ = fmt.Fprintf(a.stdout, "executable %s\n", st.Executable)
	n := st.Network
	_, _ = fmt.Fprintf(a.stdout, "usage: fetched %s, attempted %s, status %d, retry after %s\n", stamp(n.UsageFetchedAt), stamp(n.UsageAttemptAt), n.UsageStatus, stamp(n.UsageRetryAfter))
	_, _ = fmt.Fprintf(a.stdout, "health: fetched %s, attempted %s\n", stamp(n.HealthFetchedAt), stamp(n.HealthAttemptAt))
	if st.BadVersion != "" {
		_, _ = fmt.Fprintf(a.stdout, "bad version (never installed again): %s\n", st.BadVersion)
	}
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

// stamp prints an instant, or "never".
func stamp(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	return t.Local().Format(time.RFC3339)
}
