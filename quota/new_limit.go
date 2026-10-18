// Written once from the design; this file is yours. The shells in
// design_gen.go delegate to what it holds; a body that still answers
// errNotImplemented is a stub.

package quota

import (
	"time"
)

// newLimit clamps the percentage so an upstream overshoot renders as full
// and a negative value cannot invert a bar.
func newLimit(kind Kind, label string, percent int, resetsAt time.Time, window time.Duration, source Source) Limit {
	return Limit{
		Kind:     kind,
		Label:    label,
		Percent:  min(max(percent, 0), maxPercent),
		ResetsAt: resetsAt,
		Window:   window,
		Active:   true,
		Source:   source,
	}
}
