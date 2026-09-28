// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package collector

import (
	"errors"
	"testing"
)

func updating(r *releases) CheckUpdate { return NewCheckUpdate(&clock{now: t0}, r, r) }

func givenCheckUpdateInstallsAndProbesANewerRelease(t *testing.T) (CheckUpdate, CheckUpdateInput, func(*testing.T, CheckUpdateOutput)) {
	t.Helper()
	r := &releases{latest: "v1.3.0"}
	return updating(r), CheckUpdateInput{CurrentVersion: "v1.2.0"}, func(t *testing.T, out CheckUpdateOutput) {
		if !out.Installed || out.Notice.Version != "v1.3.0" || r.stored.Installed != "v1.3.0" || !r.stored.CheckedAt.Equal(t0) {
			t.Errorf("out = %+v, stored %+v", out, r.stored)
		}
	}
}

func givenCheckUpdateAsksNothingWhenDisabled(t *testing.T) (CheckUpdate, CheckUpdateInput, func(*testing.T, CheckUpdateOutput)) {
	t.Helper()
	r := &releases{latest: "v9.0.0"}
	return updating(r), CheckUpdateInput{CurrentVersion: "v1.2.0", Disabled: true}, func(t *testing.T, out CheckUpdateOutput) {
		if out.Installed || r.total() != 0 {
			t.Errorf("disabled, yet %v", r.n)
		}
	}
}

func givenCheckUpdateNeverInstallsTheBadVersionAgain(t *testing.T) (CheckUpdate, CheckUpdateInput, func(*testing.T, CheckUpdateOutput)) {
	t.Helper()
	r := &releases{latest: "v1.3.0"}
	r.stored.BadVersion = "v1.3.0"
	return updating(r), CheckUpdateInput{CurrentVersion: "v1.2.0"}, func(t *testing.T, out CheckUpdateOutput) {
		if out.Installed || r.n["install"] != 0 {
			t.Errorf("the bad version was installed: %v", r.n)
		}
	}
}

func givenCheckUpdateRollsBackAReleaseThatFailsItsProbe(t *testing.T) (CheckUpdate, CheckUpdateInput, func(*testing.T, CheckUpdateOutput)) {
	t.Helper()
	r := &releases{latest: "v1.3.0", probeErr: errors.New("exit 2")}
	// The refusal is reported as an error; the scenario checks what it left
	uc := updating(r)
	if _, err := uc.Execute(t.Context(), CheckUpdateInput{CurrentVersion: "v1.2.0"}); err == nil {
		t.Fatal("a failed probe was not reported")
	}
	if r.n["rollback"] != 1 || r.stored.BadVersion != "v1.3.0" {
		t.Fatalf("no rollback or no bad_version: %v, %+v", r.n, r.stored)
	}
	// The next check, an hour later, installs nothing
	r.stored.CheckedAt = t0.Add(-2 * 3600e9)
	return uc, CheckUpdateInput{CurrentVersion: "v1.2.0"}, func(t *testing.T, out CheckUpdateOutput) {
		if out.Installed || r.n["install"] != 1 {
			t.Errorf("the rolled-back release came back: %v", r.n)
		}
	}
}
