// Written once from the design; this file is yours. The shells in
// design_gen.go delegate to what it holds; a body that still answers
// errNotImplemented is a stub.

package ipc

import "encoding/hex"

// hash digests every field length-prefixed, so that moving a byte from one
// field to the next changes the digest (a separator byte could itself
// appear in a field).
func (k Key) hash() string {
	sum := digest(k.SessionID, k.TranscriptPath, k.SessionDir, k.TaskListID)
	return hex.EncodeToString(sum[:12])
}
