// Written once from the design; this file is yours. The shells in
// gen_design.go delegate to what it holds; a body that still answers
// errNotImplemented is a stub.

package quota

// newProgress computes a context usage percentage, capped at 100.
func newProgress(totalTokens, contextSize int) Progress {
	// A window of unknown size holds nothing
	if contextSize <= 0 {
		return Progress{}
	}
	percent := min(totalTokens*maxPercent/contextSize, maxPercent)
	// Negative token counts are upstream noise, not a negative usage
	return Progress{Percent: max(percent, 0)}
}
