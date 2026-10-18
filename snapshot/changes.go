// Written once from the design; this file is yours. The shells in
// design_gen.go delegate to what it holds; a body that still answers
// errNotImplemented is a stub.

package snapshot

func (c Changes) hasAdded() bool { return c.Added > 0 }

func (c Changes) hasChanges() bool { return c.Added > 0 || c.Removed > 0 }

func (c Changes) hasRemoved() bool { return c.Removed > 0 }
