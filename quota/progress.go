// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package quota

// level maps the percentage to its severity.
func (p Progress) level() Level {
	switch {
	case p.Percent < thresholdMedium:
		return LevelLow
	case p.Percent < thresholdHigh:
		return LevelMedium
	case p.Percent < thresholdCritical:
		return LevelHigh
	default:
		return LevelCritical
	}
}
