package sysinfo

import (
	"context"
	"runtime"
	"testing"

	"github.com/kitsunium/statusline/snapshot"
)

func TestInfoNamesTheOS(t *testing.T) {
	info, err := NewSystem().Info(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]snapshot.OS{"linux": snapshot.OSLinux, "darwin": snapshot.OSDarwin, "windows": snapshot.OSWindows}
	if w, ok := want[runtime.GOOS]; ok && info.OS != w {
		t.Errorf("OS = %d on %s, want %d", info.OS, runtime.GOOS, w)
	}
	if info.IsDocker != isDocker() {
		t.Error("IsDocker disagrees with the probe")
	}
}
