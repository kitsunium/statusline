// Written once from the design; this file is yours. The shells in
// gen_design.go delegate to what it holds; a body that still answers
// errNotImplemented is a stub.

package snapshot

// isInRepo: a branch name is only found inside a work tree.
func (s GitStatus) isInRepo() bool { return s.Branch != "" }
