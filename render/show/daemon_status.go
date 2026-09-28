// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package show

import "context"

// execute: no daemon is an answer, not an error.
func (h *DaemonStatusHandler) execute(ctx context.Context, _ DaemonStatusInput) (DaemonStatusOutput, error) {
	st, err := h.daemonControl.Status(ctx)
	if err != nil {
		return DaemonStatusOutput{}, nil
	}
	return DaemonStatusOutput{Running: true, Status: st}, nil
}
