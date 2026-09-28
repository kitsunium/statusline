// Command statusline is one executable with two process roles (D5, D22):
// `statusline` is the client the host runs on every redraw, `statusline
// daemon` the per-instance collector the client starts.
//
// This file stands in for the entry kit generates (main_gen.go) until
// `kit gen` is available.
package main

import (
	"os"

	"github.com/kitsunium/statusline/roles/client"
	"github.com/kitsunium/statusline/roles/daemon"
)

// Set at release time with -ldflags "-X main.version=… -X main.vendorKey=…".
// An empty version is a development build: it never updates itself.
var (
	version   string
	vendorKey string
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "daemon" {
		os.Exit(daemon.Main(daemon.Build{Version: version, VendorKey: vendorKey}))
	}
	os.Exit(client.Main(client.Build{Version: version}))
}
