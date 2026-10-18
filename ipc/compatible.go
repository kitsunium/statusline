// Written once from the design; this file is yours. The shells in
// design_gen.go delegate to what it holds; a body that still answers
// errNotImplemented is a stub.

package ipc

import "strings"

// compatible accepts any minor revision of the same major.
func compatible(protocol string) bool {
	name, major, ok := strings.Cut(protocol, "/v")
	if !ok || name != "statusline.ipc" {
		return false
	}
	head, _, _ := strings.Cut(major, ".")
	return head == "1"
}
