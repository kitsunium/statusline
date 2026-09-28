package releases

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/kitsunium/sdk/pkg/v1/selfupdate"

	"github.com/kitsunium/statusline/collect/state"
)

const (
	// owner and repo host the releases.
	owner string = "kitsunium"
	repo  string = "statusline"
	// prevSuffix names the previous binary kept for a rollback.
	prevSuffix string = ".prev"
	// probeTimeout bounds the new binary's --version.
	probeTimeout time.Duration = 5 * time.Second
	// execPerm is the mode of a restored binary.
	execPerm os.FileMode = 0o755
)

// errNotSigned says the build carries no vendor key: nothing is installed.
var errNotSigned = errors.New("statusline_update_unsigned_build: no vendor key, nothing is installed")

// updater is the part of the SDK service used here.
type updater interface {
	CheckForUpdate() (selfupdate.Update, error)
	Upgrade() (selfupdate.Update, error)
}

func newSource(cfg Config) *Source {
	svc := selfupdate.New(cfg.Version, selfupdate.Source{Owner: owner, StableRepo: repo, Product: repo})
	if len(cfg.VendorKey) > 0 {
		svc = svc.WithVendorKey(cfg.VendorKey)
	}
	return &Source{cfg: cfg, svc: svc}
}

func (s *Source) latest(_ context.Context) (state.Release, error) {
	info, err := s.svc.CheckForUpdate()
	if err != nil {
		return state.Release{}, err
	}
	if !info.Available {
		return state.Release{}, nil
	}
	return state.Release{Version: info.LatestVersion}, nil
}

// install copies the running binary aside first: the SDK's replacement is
// atomic but keeps nothing to return to.
func (s *Source) install(_ context.Context, rel state.Release) error {
	if len(s.cfg.VendorKey) == 0 {
		return errNotSigned
	}
	if err := copyFile(s.cfg.Executable, s.cfg.Executable+prevSuffix); err != nil {
		return fmt.Errorf("keep previous binary: %w", err)
	}
	info, err := s.svc.Upgrade()
	if err != nil {
		return err
	}
	if info.LatestVersion != rel.Version {
		return fmt.Errorf("installed %s, expected %s", info.LatestVersion, rel.Version)
	}
	return nil
}

// probe accepts the new binary only when it runs and names itself.
func (s *Source) probe(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, s.cfg.Executable, "--version").Output()
	if err != nil {
		return fmt.Errorf("probe: %w", err)
	}
	if !strings.HasPrefix(string(out), "statusline ") {
		return fmt.Errorf("probe: unexpected answer %q", strings.TrimSpace(string(out)))
	}
	return nil
}

func (s *Source) rollback(_ context.Context) error {
	return os.Rename(s.cfg.Executable+prevSuffix, s.cfg.Executable)
}

// copyFile writes dst through a temporary file next to it.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".statusline-prev-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := io.Copy(tmp, in); err != nil {
		_ = tmp.Close()
		_ = os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(name)
		return err
	}
	if err := os.Chmod(name, execPerm); err != nil {
		_ = os.Remove(name)
		return err
	}
	return os.Rename(name, dst)
}
