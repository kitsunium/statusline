// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package snapshot

// withSource tags in place: each source slice is built fresh for one call.
func (s MCPServers) withSource(src MCPSource) MCPServers {
	for i := range s {
		s[i].Source = src
	}
	return s
}

// withBusy lights the servers whose tool-name key is being called. A key no
// configuration names is still a server in use: it is appended, lit,
// without a scope, and counted enabled.
func (s MCPServers) withBusy(keys []string) MCPServers {
	out := make(MCPServers, len(s), len(s)+len(keys))
	copy(out, s)
	for _, key := range keys {
		idx := indexOfKey(out, key)
		if idx < 0 {
			out = append(out, MCPServer{Name: bareName(key), Enabled: true})
			idx = len(out) - 1
		}
		out[idx].Busy = true
	}
	return out
}
