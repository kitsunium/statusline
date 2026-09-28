package statefile

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kitsunium/statusline/collect/state"
	"github.com/kitsunium/statusline/ipc"
	"github.com/kitsunium/statusline/snapshot"
)

func TestRoundTrips(t *testing.T) {
	inst := ipc.Locate(ipc.LocateInput{RuntimeDir: t.TempDir(), UID: 1000, ConfigDir: "/c", Executable: "/e"})
	s := New(inst)

	n, err := s.LoadNetwork()
	if err != nil || n.UsageDue(time.Now()) != true {
		t.Fatalf("LoadNetwork() before any save = %+v, %v", n, err)
	}
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	saved := state.Network{UsageRetryAfter: at, UsageStatus: 429, HealthFetchedAt: at, Health: snapshot.HealthDegraded}
	if err := s.SaveNetwork(saved); err != nil {
		t.Fatal(err)
	}
	got, err := s.LoadNetwork()
	if err != nil || !got.UsageRetryAfter.Equal(at) || got.UsageStatus != 429 || got.Health != snapshot.HealthDegraded {
		t.Errorf("LoadNetwork() = %+v, %v", got, err)
	}

	if err := s.SaveUpdate(state.Update{BadVersion: "v9.9.9"}); err != nil {
		t.Fatal(err)
	}
	if u, err := s.LoadUpdate(); err != nil || u.BadVersion != "v9.9.9" {
		t.Errorf("LoadUpdate() = %+v, %v", u, err)
	}

	key := ipc.Key{SessionID: "s", SessionDir: "/w"}
	if err := s.SaveSnapshot(key, snapshot.Snapshot{WorkDir: "/w", Git: snapshot.GitStatus{Branch: "main"}}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(inst.CachePath(key))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != filePerm {
		t.Errorf("cache mode = %v, want %v", info.Mode().Perm(), filePerm)
	}
	if dir, _ := os.Stat(filepath.Dir(inst.CachePath(key))); dir.Mode().Perm() != dirPerm {
		t.Errorf("cache dir mode = %v, want %v", dir.Mode().Perm(), dirPerm)
	}
	leftovers, _ := filepath.Glob(filepath.Join(inst.Cache, "*.json.*"))
	if len(leftovers) != 0 {
		t.Errorf("temporary files left: %v", leftovers)
	}
}

func TestCorruptStateIsAnError(t *testing.T) {
	inst := ipc.Locate(ipc.LocateInput{RuntimeDir: t.TempDir(), UID: 1000, ConfigDir: "/c", Executable: "/e"})
	if err := os.MkdirAll(inst.State, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inst.State, networkFile), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := New(inst).LoadNetwork(); err == nil {
		t.Error("a corrupt state file was read as empty without a word")
	}
}
