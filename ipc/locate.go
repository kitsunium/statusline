// Written once from the design; this file is yours. The shells in
// gen_design.go delegate to what it holds; a body that still answers
// errNotImplemented is a stub.

package ipc

import "path/filepath"

// locate keeps the socket the framework placed and puts the instance's own
// files beside it, in a directory named after the same key.
func locate(in LocateInput) Instance {
	dir := filepath.Join(in.RuntimeDir, "statusline-"+in.Key)
	return Instance{
		Dir:       dir,
		Socket:    in.Socket,
		Cache:     filepath.Join(dir, "cache"),
		State:     filepath.Join(dir, "state"),
		Log:       filepath.Join(dir, "daemon.log"),
		PID:       filepath.Join(dir, "daemon.pid"),
		Heartbeat: filepath.Join(dir, "heartbeat"),
	}
}
