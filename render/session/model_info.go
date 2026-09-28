// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package session

func (m ModelInfo) fullName() string {
	if m.Version == "" {
		return m.Name
	}
	return m.Name + " " + m.Version
}

// shortName drops parenthesised qualifiers such as "(1M context)".
func (m ModelInfo) shortName() string {
	if m.Version == "" {
		return m.Name
	}
	version := m.Version
	for idx, ch := range version {
		if ch == ' ' || ch == '(' {
			version = version[:idx]
			break
		}
	}
	if version == "" {
		return m.Name
	}
	return m.Name + " " + version
}
