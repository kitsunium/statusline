// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package show

import "testing"

func givenStopDaemonStopsARunningDaemon(t *testing.T) (StopDaemon, StopDaemonInput, func(*testing.T, StopDaemonOutput)) {
	t.Helper()
	c := &control{running: true}
	return NewStopDaemon(c), StopDaemonInput{}, func(t *testing.T, out StopDaemonOutput) {
		if !out.WasRunning || !c.stopped {
			t.Errorf("out = %+v, stopped %v", out, c.stopped)
		}
	}
}

func givenStopDaemonStopsNothingWithoutADaemon(t *testing.T) (StopDaemon, StopDaemonInput, func(*testing.T, StopDaemonOutput)) {
	t.Helper()
	return NewStopDaemon(&control{}), StopDaemonInput{}, func(t *testing.T, out StopDaemonOutput) {
		if out.WasRunning {
			t.Errorf("out = %+v", out)
		}
	}
}
