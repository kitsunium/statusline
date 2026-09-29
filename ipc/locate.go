// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package ipc

import "path/filepath"

// daemonSocketPrefix is the framework's name for the daemon's listener:
// <service>-<listener>-<key>.sock, service and listener both "daemon".
const daemonSocketPrefix string = "daemon-daemon-"

// locate places the socket where the framework's listener puts it and the
// instance's own files beside it, in a directory named after the same key.
func locate(in LocateInput) Instance {
	dir := filepath.Join(in.RuntimeDir, "statusline-"+in.Key)
	return Instance{
		Dir:       dir,
		Socket:    filepath.Join(in.RuntimeDir, daemonSocketPrefix+in.Key+".sock"),
		Cache:     filepath.Join(dir, "cache"),
		State:     filepath.Join(dir, "state"),
		Log:       filepath.Join(dir, "daemon.log"),
		PID:       filepath.Join(dir, "daemon.pid"),
		Heartbeat: filepath.Join(dir, "heartbeat"),
	}
}
