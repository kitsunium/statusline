// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package show

import "context"

// execute: stopping a daemon that is not running is not an error either.
func (h *StopDaemonHandler) execute(ctx context.Context, _ StopDaemonInput) (StopDaemonOutput, error) {
	if err := h.daemonControl.Stop(ctx); err != nil {
		return StopDaemonOutput{}, nil
	}
	return StopDaemonOutput{WasRunning: true}, nil
}
