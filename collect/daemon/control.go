package daemon

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	sdkipc "github.com/kitsunium/sdk/pkg/v1/ipc"

	"github.com/kitsunium/statusline/ipc"
)

// controlBudget bounds a status or stop exchange.
const controlBudget time.Duration = 2 * time.Second

// Refusals of `statusline daemon <command>`.
var (
	errNoDaemon     = errors.New("no daemon running")
	errUnknownUsage = errors.New("usage: statusline daemon [status|stop]")
)

// control runs `statusline daemon status|stop`: it asks the instance's
// daemon through the contract, as any client would, and prints. `status`
// fails without a daemon, for scripts; `stop` does not.
func (a *Listener) control(ctx context.Context, args []string) error {
	if len(args) != 1 || (args[0] != "status" && args[0] != "stop") {
		return errUnknownUsage
	}
	out := a.stdout
	if out == nil {
		out = os.Stdout
	}
	if args[0] == "stop" {
		if _, err := a.ask(ctx, ipc.OpStop); err != nil {
			_, _ = fmt.Fprintln(out, "no daemon running")
			return nil
		}
		_, _ = fmt.Fprintln(out, "daemon stopped")
		return nil
	}
	resp, err := a.ask(ctx, ipc.OpStatus)
	// The role's main prints the refusal and exits non-zero
	if err != nil || resp.Status == nil {
		return errNoDaemon
	}
	printStatus(out, *resp.Status)
	return nil
}

// ask sends one operation to the instance's daemon.
func (a *Listener) ask(ctx context.Context, op string) (ipc.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, controlBudget)
	defer cancel()
	conn, err := sdkipc.Dial(ctx, sdkipc.Config{Path: a.instance.Socket})
	if err != nil {
		return ipc.Response{}, err
	}
	defer func() { _ = conn.Close() }()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	var hello ipc.Hello
	if err := ipc.WriteFrame(conn, ipc.Hello{Protocol: ipc.Protocol, Version: a.version, PID: os.Getpid()}); err != nil {
		return ipc.Response{}, err
	}
	if err := ipc.ReadFrame(conn, &hello); err != nil {
		return ipc.Response{}, err
	}
	if err := ipc.WriteFrame(conn, ipc.Request{Op: op}); err != nil {
		return ipc.Response{}, err
	}
	var resp ipc.Response
	return resp, ipc.ReadFrame(conn, &resp)
}

// printStatus says what the daemon is doing.
func printStatus(w io.Writer, st ipc.Status) {
	_, _ = fmt.Fprintf(w, "daemon %s pid %d, up %s, %d session(s)\n", st.Version, st.PID, time.Since(st.StartedAt).Round(time.Second), st.Sessions)
	_, _ = fmt.Fprintf(w, "executable %s\n", st.Executable)
	n := st.Network
	_, _ = fmt.Fprintf(w, "usage: fetched %s, attempted %s, status %d, retry after %s\n", stamp(n.UsageFetchedAt), stamp(n.UsageAttemptAt), n.UsageStatus, stamp(n.UsageRetryAfter))
	_, _ = fmt.Fprintf(w, "health: fetched %s, attempted %s\n", stamp(n.HealthFetchedAt), stamp(n.HealthAttemptAt))
	if st.BadVersion != "" {
		_, _ = fmt.Fprintf(w, "bad version (never installed again): %s\n", st.BadVersion)
	}
}

// stamp prints an instant, or "never".
func stamp(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	return t.Local().Format(time.RFC3339)
}
