// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package ipc

import (
	"encoding/hex"
	"path/filepath"
	"strconv"
)

// locate puts the instance under a private per-user directory; the
// instance's own directory is named after the digest of what identifies
// it, so two binaries at two paths, or two configuration directories,
// never share a daemon.
func locate(in LocateInput) Instance {
	sum := digest(in.ConfigDir, in.Executable)
	dir := filepath.Join(in.RuntimeDir, "statusline-"+strconv.Itoa(in.UID), hex.EncodeToString(sum[:8]))
	return Instance{
		Dir:       dir,
		Socket:    filepath.Join(dir, "daemon.sock"),
		Lock:      filepath.Join(dir, "daemon.lock"),
		Cache:     filepath.Join(dir, "cache"),
		State:     filepath.Join(dir, "state"),
		Log:       filepath.Join(dir, "daemon.log"),
		PID:       filepath.Join(dir, "daemon.pid"),
		Heartbeat: filepath.Join(dir, "heartbeat"),
	}
}
