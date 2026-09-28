package releases

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/kitsunium/sdk/pkg/v1/selfupdate"

	"github.com/kitsunium/statusline/collect/state"
)

// fakeUpdater stands for the SDK service: Upgrade writes a new binary.
type fakeUpdater struct {
	exe     string
	latest  string
	script  string
	upgrade error
}

func (f *fakeUpdater) CheckForUpdate() (selfupdate.Update, error) {
	return selfupdate.Update{CurrentVersion: "v1.0.0", LatestVersion: f.latest, Available: f.latest != ""}, nil
}

func (f *fakeUpdater) Upgrade() (selfupdate.Update, error) {
	if f.upgrade != nil {
		return selfupdate.Update{}, f.upgrade
	}
	if err := os.WriteFile(f.exe, []byte(f.script), 0o755); err != nil {
		return selfupdate.Update{}, err
	}
	return selfupdate.Update{LatestVersion: f.latest, Available: true}, nil
}

func setup(t *testing.T, script string) (*Source, *fakeUpdater, string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell scripts stand for binaries")
	}
	exe := filepath.Join(t.TempDir(), "statusline")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\necho statusline v1.0.0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	fake := &fakeUpdater{exe: exe, latest: "v1.1.0", script: script}
	return &Source{cfg: Config{Version: "v1.0.0", Executable: exe, VendorKey: []byte("key")}, svc: fake}, fake, exe
}

func TestInstallKeepsPreviousAndProbes(t *testing.T) {
	src, _, exe := setup(t, "#!/bin/sh\necho statusline v1.1.0\n")
	ctx := context.Background()
	rel, err := src.Latest(ctx)
	if err != nil || rel.Version != "v1.1.0" {
		t.Fatalf("Latest() = %+v, %v", rel, err)
	}
	if err := src.Install(ctx, rel); err != nil {
		t.Fatal(err)
	}
	prev, err := os.ReadFile(exe + prevSuffix)
	if err != nil || string(prev) != "#!/bin/sh\necho statusline v1.0.0\n" {
		t.Errorf("<bin>.prev = %q, %v", prev, err)
	}
	if err := src.Probe(ctx); err != nil {
		t.Errorf("Probe() = %v", err)
	}
}

func TestBadBinaryRollsBack(t *testing.T) {
	src, _, exe := setup(t, "#!/bin/sh\nexit 3\n")
	ctx := context.Background()
	if err := src.Install(ctx, state.Release{Version: "v1.1.0"}); err != nil {
		t.Fatal(err)
	}
	if err := src.Probe(ctx); err == nil {
		t.Fatal("a failing binary passed its probe")
	}
	if err := src.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(exe); string(got) != "#!/bin/sh\necho statusline v1.0.0\n" {
		t.Errorf("after rollback the binary is %q", got)
	}
}

func TestUnsignedBuildInstallsNothing(t *testing.T) {
	src, _, exe := setup(t, "#!/bin/sh\necho statusline v1.1.0\n")
	src.cfg.VendorKey = nil
	if err := src.Install(context.Background(), state.Release{Version: "v1.1.0"}); !errors.Is(err, errNotSigned) {
		t.Errorf("Install() = %v, want errNotSigned", err)
	}
	if _, err := os.Stat(exe + prevSuffix); err == nil {
		t.Error("an unsigned build touched the disk")
	}
}

func TestLatestNothingNewer(t *testing.T) {
	src, fake, _ := setup(t, "")
	fake.latest = ""
	if rel, err := src.Latest(context.Background()); err != nil || rel.Version != "" {
		t.Errorf("Latest() = %+v, %v", rel, err)
	}
}
