// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package show

import "testing"

func givenDaemonStatusReportsARunningDaemon(t *testing.T) (DaemonStatus, DaemonStatusInput, func(*testing.T, DaemonStatusOutput)) {
	t.Helper()
	return NewDaemonStatus(&control{running: true}), DaemonStatusInput{}, func(t *testing.T, out DaemonStatusOutput) {
		if !out.Running || out.Status.Version != "v1.2.3" || out.Status.Sessions != 2 {
			t.Errorf("out = %+v", out)
		}
	}
}

func givenDaemonStatusReportsNoDaemon(t *testing.T) (DaemonStatus, DaemonStatusInput, func(*testing.T, DaemonStatusOutput)) {
	t.Helper()
	return NewDaemonStatus(&control{}), DaemonStatusInput{}, func(t *testing.T, out DaemonStatusOutput) {
		if out.Running {
			t.Errorf("out = %+v, want not running", out)
		}
	}
}
