// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package session

func effortRank(level string) (int, bool) {
	for idx, known := range effortScale {
		if known == level {
			return idx + 1, true
		}
	}
	return 0, false
}
