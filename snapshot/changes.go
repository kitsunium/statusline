// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package snapshot

func (c Changes) hasAdded() bool { return c.Added > 0 }

func (c Changes) hasChanges() bool { return c.Added > 0 || c.Removed > 0 }

func (c Changes) hasRemoved() bool { return c.Removed > 0 }
