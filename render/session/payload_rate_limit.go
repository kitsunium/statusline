// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package session

// percent prefers used_percentage, then utilization.
func (r *PayloadRateLimit) percent() (int, bool) {
	if r == nil {
		return 0, false
	}
	if r.UsedPercentage != nil {
		return int(*r.UsedPercentage), true
	}
	if r.Utilization != nil {
		return int(*r.Utilization), true
	}
	return 0, false
}
