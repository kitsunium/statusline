// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

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
