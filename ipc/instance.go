// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package ipc

import "path/filepath"

// cachePath names the key's snapshot file in the instance's cache.
func (i Instance) cachePath(key Key) string {
	return filepath.Join(i.Cache, key.hash()+".json")
}
