package quota

import "strings"

// timed lists session, weekly then scoped, keeping only the valid ones.
func (s Set) timed() []Limit {
	limits := make([]Limit, 0, len(s.Scoped)+2)
	if s.Session.IsValid() {
		limits = append(limits, s.Session)
	}
	if s.Weekly.IsValid() {
		limits = append(limits, s.Weekly)
	}
	for _, scoped := range s.Scoped {
		if scoped.IsValid() {
			limits = append(limits, scoped)
		}
	}
	return limits
}

// hasTimed reports whether any time-boxed quota is present.
func (s Set) hasTimed() bool {
	return len(s.timed()) > 0
}

// scopedFor keeps the scoped quotas whose label names the model in use:
// showing the Fable quota while working in Opus is noise that crowds out
// the numbers that matter.
func (s Set) scopedFor(modelName string) []Limit {
	lowered := strings.ToLower(modelName)
	matching := make([]Limit, 0, len(s.Scoped))
	for _, scoped := range s.Scoped {
		if !scoped.IsValid() || !strings.Contains(lowered, strings.ToLower(scoped.Label)) {
			continue
		}
		matching = append(matching, scoped)
	}
	return matching
}
