// Written once from the design; this file is yours. The shells in
// gen_design.go delegate to what it holds; a body that still answers
// errNotImplemented is a stub.

package ipc

import "path/filepath"

// cachePath names the key's snapshot file in the instance's cache.
func (i Instance) cachePath(key Key) string {
	return filepath.Join(i.Cache, key.hash()+".json")
}
