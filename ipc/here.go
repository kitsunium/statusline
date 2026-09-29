package ipc

import (
	"github.com/kitsunium/sdk/framework/kit"
	sdkipc "github.com/kitsunium/sdk/pkg/v1/ipc"
)

const (
	// product names the runtime directory that holds the instances' sockets.
	product string = "statusline"
	// daemon is the daemon's service and its listener (design/collect.yaml).
	daemon string = "daemon"
)

// scopes are the daemon's singletonPer and socketPer (design/product.yaml,
// collect.yaml), evaluated in this process.
var scopes = []kit.Scope{kit.PerUID, kit.PerExecutable, kit.PerEnv("CLAUDE_CONFIG_DIR", "~/.claude")}

// here asks the framework for the key and the socket, so the client dials
// the path the listener opened without importing its declaration.
func here() (Instance, error) {
	key, err := kit.ScopeKey(scopes...)
	if err != nil {
		return Instance{}, err
	}
	socket, err := kit.SocketPathFor(product, daemon, daemon, scopes...)
	if err != nil {
		return Instance{}, err
	}
	return locate(LocateInput{RuntimeDir: sdkipc.RuntimeDir(product), Key: key, Socket: socket}), nil
}
