// Written once from the design; this file is yours. The shells in
// design_gen.go delegate to what it holds; a body that still answers
// errNotImplemented is a stub.

package snapshot

import "strings"

// toolKey replaces every character outside [A-Za-z0-9_-] by an underscore.
func toolKey(name string) string {
	return strings.Map(func(r rune) rune {
		if r == '_' || r == '-' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, name)
}
