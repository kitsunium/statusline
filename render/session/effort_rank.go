// Written once from the design; this file is yours. The shells in
// design_gen.go delegate to what it holds; a body that still answers
// errNotImplemented is a stub.

package session

func effortRank(level string) (int, bool) {
	for idx, known := range effortScale {
		if known == level {
			return idx + 1, true
		}
	}
	return 0, false
}
