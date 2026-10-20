// Written once from the design; this file is yours. The shells in
// gen_design.go delegate to what it holds; a body that still answers
// errNotImplemented is a stub.

package collector

import "context"

// execute serves the last figures; the health is unknown past HealthMaxAge.
func (h *LatestNetworkHandler) execute(ctx context.Context, _ *LatestNetworkInput) (LatestNetworkOutput, error) {
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
