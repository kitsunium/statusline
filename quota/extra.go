// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package quota

// isValid: credits switched off carry no useful signal.
func (e Extra) isValid() bool {
	return e.Enabled && e.Source != SourceNone
}
