package releases

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kitsunium/sdk/pkg/v1/errs"
	"github.com/kitsunium/sdk/pkg/v1/selfupdate"

	"github.com/kitsunium/statusline/collect/state"
)

const (
	// owner and repo host the releases.
	owner string = "kitsunium"
	repo  string = "statusline"
	// signatureDomain is what every release signature covers besides the
	// manifest: a key that signs other documents cannot sign a release.
	signatureDomain string = "kitsunium/statusline release"
	// probeTimeout bounds the new binary's --version.
	probeTimeout time.Duration = 5 * time.Second
)

// probeArgs is what the new binary must answer, exit 0, before it stays.
var probeArgs = []string{"--version"}

// errNotSigned says the build carries no vendor key: nothing is installed.
var errNotSigned = errors.New("statusline_update_unsigned_build: no vendor key, nothing is installed")

// updater is the part of the SDK service used here.
type updater interface {
	CheckForUpdate() (selfupdate.Update, error)
	Upgrade() (selfupdate.Update, error)
}

// config is what a build knows about itself.
type config struct {
	// Version is the running release; "dev" for a development build.
	Version string
	// VendorKeys are the ed25519 keys releases are signed with, newest first.
	VendorKeys [][]byte
}

// releases is the build's identity and the SDK's update service.
type releases struct {
	cfg config
	svc updater
}

// newReleases takes the build's identity: its version, and the base64
// ed25519 keys releases are signed with, comma-separated
// (-ldflags -X main.vendorKey=…); without a key nothing is ever installed.
func newReleases(version string, vendorKey string) *Releases {
	var keys [][]byte
	for _, k := range strings.Split(vendorKey, ",") {
		if key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(k)); err == nil && len(key) > 0 {
			keys = append(keys, key)
		}
	}
	return newReleasesWith(config{Version: version, VendorKeys: keys})
}

// newReleasesWith builds the SDK service: automatic consent declared by
// the product (D11), never any elevation, a signature domain, and the probe
// that keeps <bin>.prev and puts it back when the new binary fails.
func newReleasesWith(cfg config) *Releases {
	svc := selfupdate.New(cfg.Version, selfupdate.Source{Owner: owner, StableRepo: repo, Product: repo}).
		WithSignatureDomain(signatureDomain).
		WithProbe(probeArgs, probeTimeout).
		WithAutomaticConsent().
		WithoutElevation()
	if len(cfg.VendorKeys) > 0 {
		svc = svc.WithVendorKeys(cfg.VendorKeys...)
	}
	return &Releases{releases{cfg: cfg, svc: svc}}
}

func (s *Releases) latest(_ context.Context) (state.Release, error) {
	info, err := s.svc.CheckForUpdate()
	if err != nil {
		return state.Release{}, err
	}
	if !info.Available {
		return state.Release{}, nil
	}
	return state.Release{Version: info.LatestVersion}, nil
}

// install lets the SDK replace, probe and roll back; a failed probe is the
// product's state.ErrProbeFailed, so that the version is recorded bad.
func (s *Releases) install(_ context.Context, rel state.Release) error {
	if len(s.cfg.VendorKeys) == 0 {
		return errNotSigned
	}
	info, err := s.svc.Upgrade()
	switch {
	case errs.HasCode(err, selfupdate.CodeProbeFailed):
		return fmt.Errorf("%w: %v", state.ErrProbeFailed, err)
	case err != nil:
		return err
	case info.LatestVersion != rel.Version:
		return fmt.Errorf("installed %s, expected %s", info.LatestVersion, rel.Version)
	}
	return nil
}
