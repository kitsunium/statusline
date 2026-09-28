package update_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/kitsunium/statusline/collect/state"
	"github.com/kitsunium/statusline/collect/update"
)

var t0 = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

// host records every port call and answers from its fields.
type host struct {
	calls      []string
	stored     state.Update
	latest     string
	installErr error
	probeErr   error
}

func (h *host) Now() time.Time { return t0 }

func (h *host) LoadUpdate() (state.Update, error) {
	h.calls = append(h.calls, "load")
	return h.stored, nil
}

func (h *host) SaveUpdate(u state.Update) error {
	h.calls = append(h.calls, "save")
	h.stored = u
	return nil
}

func (h *host) Latest(context.Context) (state.Release, error) {
	h.calls = append(h.calls, "latest")
	return state.Release{Version: h.latest}, nil
}

func (h *host) Install(_ context.Context, rel state.Release) error {
	h.calls = append(h.calls, "install:"+rel.Version)
	return h.installErr
}

func (h *host) Probe(context.Context) error {
	h.calls = append(h.calls, "probe")
	return h.probeErr
}

func (h *host) Rollback(context.Context) error {
	h.calls = append(h.calls, "rollback")
	return nil
}

func run(h *host, in update.UpdateInput) (update.UpdateOutput, error) {
	return update.New(update.Deps{Clock: h, Store: h, Releases: h}).Execute(context.Background(), in)
}

// TestSequenceSelfUpdate pins design/sequences/self-update.yaml.
func TestSequenceSelfUpdate(t *testing.T) {
	h := &host{latest: "v1.3.0"}
	out, err := run(h, update.UpdateInput{CurrentVersion: "v1.2.0"})
	if err != nil || !out.Installed || out.Notice.Version != "v1.3.0" {
		t.Fatalf("Execute() = %+v, %v", out, err)
	}
	if got, want := strings.Join(h.calls, ","), "load,latest,install:v1.3.0,probe,save"; got != want {
		t.Errorf("calls = %s, want %s", got, want)
	}
	if h.stored.Installed != "v1.3.0" || !h.stored.CheckedAt.Equal(t0) {
		t.Errorf("stored = %+v", h.stored)
	}
}

func TestUpToDate(t *testing.T) {
	for _, latest := range []string{"", "v1.2.0", "v1.1.9"} {
		h := &host{latest: latest}
		if out, err := run(h, update.UpdateInput{CurrentVersion: "v1.2.0"}); err != nil || out.Installed {
			t.Errorf("latest %q: %+v, %v", latest, out, err)
		}
		if got := strings.Join(h.calls, ","); got != "load,latest,save" {
			t.Errorf("latest %q: calls %s", latest, got)
		}
	}
}

func TestDisabledAsksNothing(t *testing.T) {
	for _, in := range []update.UpdateInput{
		{CurrentVersion: "v1.2.0", Disabled: true},
		{CurrentVersion: ""},
		{CurrentVersion: "dev"},
	} {
		h := &host{latest: "v9.0.0"}
		if out, _ := run(h, in); out.Installed || len(h.calls) != 0 {
			t.Errorf("%+v: installed %v, calls %v", in, out.Installed, h.calls)
		}
	}
}

func TestNotDue(t *testing.T) {
	h := &host{latest: "v9.0.0", stored: state.Update{CheckedAt: t0.Add(-time.Minute)}}
	if _, _ = run(h, update.UpdateInput{CurrentVersion: "v1.0.0"}); strings.Join(h.calls, ",") != "load" {
		t.Errorf("checked within the hour: %v", h.calls)
	}
}

func TestBadVersionIsNeverInstalledAgain(t *testing.T) {
	h := &host{latest: "v1.3.0", stored: state.Update{BadVersion: "v1.3.0"}}
	if out, _ := run(h, update.UpdateInput{CurrentVersion: "v1.2.0"}); out.Installed {
		t.Error("the bad version was installed")
	}
	for _, c := range h.calls {
		if strings.HasPrefix(c, "install") {
			t.Fatalf("install attempted: %v", h.calls)
		}
	}
}

func TestProbeFailsRollsBack(t *testing.T) {
	h := &host{latest: "v1.3.0", probeErr: errors.New("exit 2")}
	out, err := run(h, update.UpdateInput{CurrentVersion: "v1.2.0"})
	if err == nil || out.Installed {
		t.Fatalf("Execute() = %+v, %v; want a refusal", out, err)
	}
	if got, want := strings.Join(h.calls, ","), "load,latest,install:v1.3.0,probe,rollback,save"; got != want {
		t.Errorf("calls = %s, want %s", got, want)
	}
	if h.stored.BadVersion != "v1.3.0" {
		t.Errorf("bad_version = %q, want v1.3.0", h.stored.BadVersion)
	}
}

func TestInstallFailureIsCounted(t *testing.T) {
	h := &host{latest: "v1.3.0", installErr: errors.New("signature invalid")}
	if _, err := run(h, update.UpdateInput{CurrentVersion: "v1.2.0"}); err == nil {
		t.Fatal("an install failure was swallowed")
	}
	if h.stored.Failures != 1 || h.stored.BadVersion != "" {
		t.Errorf("stored = %+v", h.stored)
	}
}
