package ipc

import (
	"github.com/kitsunium/sdk/framework/kit"
	sdkipc "github.com/kitsunium/sdk/pkg/v1/ipc"
)

// daemonApp is the framework app the daemon role runs: its runtime
// directory holds the instances' sockets.
const daemonApp string = "daemon"

// here computes the key the daemon's singleton and socket take, from the
// same scopes (design/product.yaml, collect.yaml), in this process.
func here() (Instance, error) {
	key, err := kit.ScopeKey(kit.PerUID, kit.PerExecutable, kit.PerEnv("CLAUDE_CONFIG_DIR", "~/.claude"))
	if err != nil {
		return Instance{}, err
	}
	return locate(LocateInput{RuntimeDir: sdkipc.RuntimeDir(daemonApp), Key: key}), nil
}
