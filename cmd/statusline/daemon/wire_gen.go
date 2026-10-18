// Code generated from design/v1; DO NOT EDIT.
// design:digest sha256:f7116793c8d8ac76611661ce6107f6662b6eb477309df7bb89556666304dc4e6

// Package daemon — One per (UID, host configuration directory, executable),
// started by a client, stopped a minute after its last session.
package daemon

import (
	"context"
	"time"

	"github.com/kitsunium/sdk/framework/app"
	"github.com/kitsunium/sdk/framework/model"
	"github.com/kitsunium/statusline/collect/collector"
	"github.com/kitsunium/statusline/collect/credentials"
	daemonc "github.com/kitsunium/statusline/collect/daemon"
	"github.com/kitsunium/statusline/collect/gitcli"
	"github.com/kitsunium/statusline/collect/mcpconfig"
	"github.com/kitsunium/statusline/collect/releases"
	"github.com/kitsunium/statusline/collect/sessioncache"
	"github.com/kitsunium/statusline/collect/sessions"
	"github.com/kitsunium/statusline/collect/statefile"
	"github.com/kitsunium/statusline/collect/statuspage"
	"github.com/kitsunium/statusline/collect/sysinfo"
	"github.com/kitsunium/statusline/collect/taskstore"
	"github.com/kitsunium/statusline/collect/transcripts"
	"github.com/kitsunium/statusline/collect/usageapi"
	"github.com/kitsunium/statusline/collect/wallclock"
)

// App is role daemon: the framework App of daemon, profile daemon.
var App = app.NewApp("daemon", daemonc.Service).With(app.Profile(model.ProfileDaemon), app.IdleStop(time.Minute), app.SingletonPer(app.PerUID, app.PerExecutable, app.PerEnv("CLAUDE_CONFIG_DIR", "~/.claude")))

// Build is what the binary's build says: its variables, set by -ldflags -X.
type Build struct {
	Version   string
	VendorKey string
}

// Wire builds what role daemon's handlers use, and gives it to them.
func Wire(ctx context.Context, build Build) {
	clock := wallclock.NewClock()
	cache := sessioncache.NewCache()
	transcripts := transcripts.NewTranscripts()
	sessions := sessions.NewSessions()
	git := gitcli.NewGit()
	system := sysinfo.NewSystem()
	config := mcpconfig.NewConfig()
	tasks := taskstore.NewTasks()
	files := statefile.NewFiles()
	latestNetwork := collector.NewLatestNetwork(clock, files)
	collectSnapshot := collector.NewCollectSnapshot(clock, cache, transcripts, sessions, git, system, transcripts, config, tasks, latestNetwork, files)
	credentials := credentials.NewCredentials()
	usage := usageapi.NewUsage()
	statusPage := statuspage.NewStatusPage()
	refreshNetwork := collector.NewRefreshNetwork(clock, files, credentials, usage, statusPage)
	releases := releases.NewReleases(build.Version, build.VendorKey)
	checkUpdate := collector.NewCheckUpdate(clock, files, releases)
	daemonc.Wire(collectSnapshot, latestNetwork, cache, files, build.Version, refreshNetwork, clock, checkUpdate)
}
