package quota

import "strings"

// isValid: credits switched off carry no useful signal.
func (e Extra) isValid() bool {
	return e.Enabled && e.Source != SourceNone
}

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

// resolve lets stdin win wherever both sources carry a bucket: it is free,
// synchronous and always current, while the API is cached and slightly
// behind. The API only contributes the model-scoped quotas, the credit
// balance and the buckets a host build did not send. A bucket missing from
// both stays missing: rendering it as 0 % would lie about the account.
func resolve(stdin, api Set) Set {
	merged := stdin
	if !merged.Session.IsValid() && api.Session.IsValid() {
		merged.Session = api.Session
	}
	if !merged.Weekly.IsValid() && api.Weekly.IsValid() {
		merged.Weekly = api.Weekly
	}
	merged.Scoped = api.Scoped
	merged.Extra = api.Extra
	return merged
}
