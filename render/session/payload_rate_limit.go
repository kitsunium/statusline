// Written once from the design; this file is yours. The shells in
// gen_design.go delegate to what it holds; a body that still answers
// errNotImplemented is a stub.

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
