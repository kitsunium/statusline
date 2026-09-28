// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package snapshot

// withSource tags in place: each source slice is built fresh for one call.
func withSource(servers []MCPServer, source string) []MCPServer {
	for i := range servers {
		servers[i].Source = source
	}
	return servers
}
