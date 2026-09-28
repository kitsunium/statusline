package cli

import "github.com/kitsunium/statusline/ipc"

// buildVersion is the release this binary was built as.
func buildVersion() string { return ipc.BuildVersion() }
