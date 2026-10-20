// Written once from the design; this file is yours. The shells in
// gen_design.go delegate to what it holds; a body that still answers
// errNotImplemented is a stub.

package session

import "encoding/json"

// parse never fails: the renderer degrades to what it can establish alone,
// which beats a raw error on every redraw. Invalid JSON decodes nothing; a
// mistyped field leaves what encoding/json decoded around it, exactly as the
// legacy product did.
func parse(data []byte) Payload {
	var p Payload
	_ = json.Unmarshal(data, &p)
	return p
}
