// Written once from the design; this file is yours. The shells in
// design_gen.go delegate to what it holds; a body that still answers
// errNotImplemented is a stub.

package snapshot

func (b TaskBoard) isEmpty() bool { return len(b.Epics) == 0 && b.Unattributed == 0 }
