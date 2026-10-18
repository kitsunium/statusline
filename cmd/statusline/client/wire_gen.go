// Code generated from design/v1; DO NOT EDIT.
// design:digest sha256:f7116793c8d8ac76611661ce6107f6662b6eb477309df7bb89556666304dc4e6

// Package client — What the host runs on every redraw.
package client

import (
	"context"

	"github.com/kitsunium/sdk/framework/app"
	"github.com/kitsunium/sdk/framework/model"
	"github.com/kitsunium/statusline/render/cli"
	"github.com/kitsunium/statusline/render/daemonlink"
	"github.com/kitsunium/statusline/render/show"
	"github.com/kitsunium/statusline/render/systemclock"
)

// App is role client: the framework App of cli, profile cli.
var App = app.NewApp("client", cli.Service).With(app.Profile(model.ProfileCLI))

// Build is what the binary's build says: its variables, set by -ldflags -X.
type Build struct {
	Version   string
	VendorKey string
}

// Wire builds what role client's handlers use, and gives it to them.
func Wire(ctx context.Context, build Build) {
	link := daemonlink.NewLink(build.Version)
	clock := systemclock.NewClock()
	showStatusLine := show.NewShowStatusLine(link, clock)
	cli.Wire(showStatusLine, build.Version)
}
