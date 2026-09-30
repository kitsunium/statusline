// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package collector

import "context"

// execute serves the last figures; the health is unknown past HealthMaxAge.
func (h *LatestNetworkHandler) execute(ctx context.Context, _ LatestNetworkInput) (LatestNetworkOutput, error) {
	now, err := h.clock.Now(ctx)
	if err != nil {
		return LatestNetworkOutput{}, err
	}
	n, err := h.networkStore.LoadNetwork(ctx)
	if err != nil {
		return LatestNetworkOutput{}, err
	}
	return LatestNetworkOutput{Usage: n.Usage, HasUsage: n.HasUsage, Health: n.CurrentHealth(now), Network: n}, nil
}
