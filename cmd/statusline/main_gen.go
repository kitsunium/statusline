// Code generated from design/v1; DO NOT EDIT.
// design:digest sha256:f7116793c8d8ac76611661ce6107f6662b6eb477309df7bb89556666304dc4e6

// Command statusline — The status line, and the daemon that collects for it.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/kitsunium/sdk/framework/app"
	_ "github.com/kitsunium/sdk/framework/app/jobs"
	clientrole "github.com/kitsunium/statusline/cmd/statusline/client"
	daemonrole "github.com/kitsunium/statusline/cmd/statusline/daemon"
)

// Binary is the executable and its process roles (D22).
var Binary = app.NewBinary("statusline", "The status line, and the daemon that collects for it.").
	Role("client", clientrole.App).
	Role("daemon", daemonrole.App, "daemon").
	Talks("client", "daemon", "daemon/v1")

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	args := os.Args[1:]
	wire(ctx, args)
	code := Binary.Main(ctx, args)
	stop()
	os.Exit(code)
}

// wire builds what the role the arguments pick needs, before it runs.
func wire(ctx context.Context, args []string) {
	if len(args) > 0 {
		switch args[0] {
		case "daemon":
			daemonrole.Wire(ctx, daemonrole.Build{Version: version, VendorKey: vendorKey})
			return
		}
	}
	clientrole.Wire(ctx, clientrole.Build{Version: version, VendorKey: vendorKey})
}

// The build's variables: go build -ldflags "-X main.<name>=<value>".
var (
	version   = "dev"
	vendorKey = ""
)
