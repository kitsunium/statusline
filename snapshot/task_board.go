// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package snapshot

func (b TaskBoard) isEmpty() bool { return len(b.Epics) == 0 && b.Unattributed == 0 }
