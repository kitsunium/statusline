// Written once from the design; this file is yours. The shells in
// design_gen.go delegate to what it holds; a body that still answers
// errNotImplemented is a stub.

package state

import (
	"time"
)

func (u Update) due(now time.Time) bool {
	return u.CheckedAt.IsZero() || now.Sub(u.CheckedAt) >= UpdateInterval
}
