package releases

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"

	"github.com/kitsunium/sdk/pkg/v1/selfupdate"

	"github.com/kitsunium/statusline/collect/state"
)

// fakeUpdater stands for the SDK service.
type fakeUpdater struct {
	latest   string
	upgrade  error
	upgraded int
}

func (f *fakeUpdater) CheckForUpdate() (selfupdate.Update, error) {
	return selfupdate.Update{CurrentVersion: "v1.0.0", LatestVersion: f.latest, Available: f.latest != ""}, nil
}

func (f *fakeUpdater) Upgrade() (selfupdate.Update, error) {
	f.upgraded++
	if f.upgrade != nil {
		return selfupdate.Update{}, f.upgrade
	}
	return selfupdate.Update{LatestVersion: f.latest, Available: true}, nil
}

func source(fake *fakeUpdater, keys int) *Releases {
	cfg := config{Version: "v1.0.0"}
	for i := 0; i < keys; i++ {
		cfg.VendorKeys = append(cfg.VendorKeys, []byte("key"))
	}
	return &Releases{releases{cfg: cfg, svc: fake}}
}

func TestLatestAndInstall(t *testing.T) {
	fake := &fakeUpdater{latest: "v1.1.0"}
	src := source(fake, 1)
	rel, err := src.Latest(context.Background())
	if err != nil || rel.Version != "v1.1.0" {
		t.Fatalf("Latest() = %+v, %v", rel, err)
	}
	if err := src.Install(context.Background(), rel); err != nil || fake.upgraded != 1 {
		t.Errorf("Install() = %v, upgraded %d", err, fake.upgraded)
	}
	fake.latest = ""
	if rel, err := src.Latest(context.Background()); err != nil || rel.Version != "" {
		t.Errorf("nothing newer: %+v, %v", rel, err)
	}
}

func TestProbeFailureIsTheProductsError(t *testing.T) {
	src := source(&fakeUpdater{latest: "v1.1.0", upgrade: selfupdate.ProbeFailed}, 1)
	err := src.Install(context.Background(), state.Release{Version: "v1.1.0"})
	if !errors.Is(err, state.ErrProbeFailed) {
		t.Errorf("Install() = %v, want state.ErrProbeFailed", err)
	}
	other := source(&fakeUpdater{latest: "v1.1.0", upgrade: selfupdate.SignatureInvalid}, 1)
	if err := other.Install(context.Background(), state.Release{Version: "v1.1.0"}); err == nil || errors.Is(err, state.ErrProbeFailed) {
		t.Errorf("a bad signature = %v, want an error that is no probe failure", err)
	}
}

func TestUnsignedBuildInstallsNothing(t *testing.T) {
	fake := &fakeUpdater{latest: "v1.1.0"}
	if err := source(fake, 0).Install(context.Background(), state.Release{Version: "v1.1.0"}); !errors.Is(err, errNotSigned) || fake.upgraded != 0 {
		t.Errorf("Install() = %v, upgraded %d", err, fake.upgraded)
	}
}

func TestVendorKeysAreCommaSeparated(t *testing.T) {
	a, b := base64.StdEncoding.EncodeToString([]byte("first")), base64.StdEncoding.EncodeToString([]byte("second"))
	r := newReleases("v1.0.0", a+", "+b+",,!notbase64")
	if len(r.cfg.VendorKeys) != 2 || string(r.cfg.VendorKeys[1]) != "second" {
		t.Errorf("keys = %q", r.cfg.VendorKeys)
	}
	if len(newReleases("dev", "").cfg.VendorKeys) != 0 {
		t.Error("an empty key string gave keys")
	}
}
