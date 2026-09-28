// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package state

import (
	"time"
)

func (u Update) due(now time.Time) bool {
	return u.CheckedAt.IsZero() || now.Sub(u.CheckedAt) >= UpdateInterval
}
