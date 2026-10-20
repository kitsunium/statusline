// Written once from the design; this file is yours. The shells in
// gen_design.go delegate to what it holds; a body that still answers
// errNotImplemented is a stub.

package snapshot

// classifyHealth: one impaired component is degraded, two or one major
// outage is down; maintenance does not count; nothing to judge is unknown.
func classifyHealth(states []string) Health {
	if len(states) == 0 {
		return HealthUnknown
	}
	degraded := 0
	for _, state := range states {
		switch state {
		case componentMajor:
			return HealthDown
		case componentDegraded, componentPartial:
			degraded++
		}
	}
	switch {
	case degraded >= degradedForDown:
		return HealthDown
	case degraded == 1:
		return HealthDegraded
	default:
		return HealthOK
	}
}
