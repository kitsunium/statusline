// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package snapshot

// isInRepo: a branch name is only found inside a work tree.
func (s GitStatus) isInRepo() bool { return s.Branch != "" }
