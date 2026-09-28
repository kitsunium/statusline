package ipc

import (
	"bytes"
	"encoding/binary"
	"path/filepath"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// seeded logs the design seed: rapid v1.2 takes its own from -rapid.seed.
func seeded(t *testing.T, seed uint64) {
	t.Helper()
	t.Logf("design seed %d (rapid v1.2 takes its own seed from -rapid.seed)", seed)
}

func propertyInstanceCachePathInsideCache(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		inst := Locate(LocateInput{RuntimeDir: "/run/user/1", UID: 1, ConfigDir: rapid.String().Draw(t, "c"), Executable: "/e"})
		p := inst.CachePath(keyGen().Draw(t, "k"))
		if filepath.Dir(p) != inst.Cache || !strings.HasSuffix(p, ".json") {
			t.Fatalf("CachePath = %q outside %q", p, inst.Cache)
		}
	})
}

func propertyBuildVersionStable(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		if BuildVersion() != BuildVersion() {
			t.Fatal("BuildVersion changed between calls")
		}
		if v := BuildVersion(); v != "" && CompareVersions(v, "v0.0.0") < 0 {
			t.Fatalf("BuildVersion %q is neither a release nor empty", v)
		}
	})
}

func propertyCompatibleMajorOne(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		major := rapid.IntRange(0, 5).Draw(t, "major")
		minor := rapid.StringMatching(`(\.[0-9]{1,2})?`).Draw(t, "minor")
		p := "statusline.ipc/v" + string(rune('0'+major)) + minor
		if Compatible(p) != (major == 1) {
			t.Fatalf("Compatible(%q) = %v", p, Compatible(p))
		}
	})
}

func propertyHereUnderRuntimeDir(t *testing.T, seed uint64) {
	seeded(t, seed)
	dir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", dir)
	rapid.Check(t, func(rt *rapid.T) {
		cfg := "/cfg/" + rapid.StringMatching(`[a-z]{1,8}`).Draw(rt, "cfg")
		t.Setenv("CLAUDE_CONFIG_DIR", cfg)
		inst, err := Here()
		if err != nil || !strings.HasPrefix(inst.Socket, dir+string(filepath.Separator)) {
			rt.Fatalf("Here() = %+v, %v", inst, err)
		}
	})
}

func propertyWriteFrameLengthPrefixed(t *testing.T, seed uint64) {
	seeded(t, seed)
	rapid.Check(t, func(t *rapid.T) {
		req := Request{Op: rapid.String().Draw(t, "op"), Key: keyGen().Draw(t, "key")}
		var buf bytes.Buffer
		if err := WriteFrame(&buf, req); err != nil {
			t.Fatal(err)
		}
		if n := binary.BigEndian.Uint32(buf.Bytes()[:4]); int(n) != buf.Len()-4 {
			t.Fatalf("header says %d, body is %d", n, buf.Len()-4)
		}
	})
}
