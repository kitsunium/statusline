package ipc_test

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"pgregory.net/rapid"

	"github.com/kitsunium/statusline/ipc"
	"github.com/kitsunium/statusline/snapshot"
)

func keyGen() *rapid.Generator[ipc.Key] {
	return rapid.Custom(func(t *rapid.T) ipc.Key {
		return ipc.Key{
			SessionID:      rapid.String().Draw(t, "session"),
			TranscriptPath: rapid.String().Draw(t, "transcript"),
			SessionDir:     rapid.String().Draw(t, "dir"),
			TaskListID:     rapid.String().Draw(t, "list"),
		}
	})
}

// TestPropertyKeyHashStable (ipc/property/key-hash-stable).
func TestPropertyKeyHashStable(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a, b := keyGen().Draw(t, "a"), keyGen().Draw(t, "b")
		if a.Hash() != a.Hash() {
			t.Fatal("hash not stable")
		}
		if a != b && a.Hash() == b.Hash() {
			t.Fatalf("collision: %+v and %+v", a, b)
		}
		// Moving a byte across a field boundary changes the hash
		if a.SessionID != "" {
			moved := a
			moved.SessionID = a.SessionID[:len(a.SessionID)-1]
			moved.TranscriptPath = a.SessionID[len(a.SessionID)-1:] + a.TranscriptPath
			if moved.Hash() == a.Hash() {
				t.Fatal("a byte moved across fields kept the hash")
			}
		}
	})
}

// TestPropertyLocateDistinct (ipc/property/locate-distinct).
func TestPropertyLocateDistinct(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		in := ipc.LocateInput{RuntimeDir: "/run/user/1000", UID: 1000,
			ConfigDir: rapid.String().Draw(t, "cfg"), Executable: rapid.String().Draw(t, "exe")}
		other := in
		if rapid.Bool().Draw(t, "which") {
			other.ConfigDir += rapid.StringN(1, 4, -1).Draw(t, "more")
		} else {
			other.Executable += rapid.StringN(1, 4, -1).Draw(t, "more")
		}
		if ipc.Locate(in).Socket == ipc.Locate(other).Socket {
			t.Fatal("two instances share a socket")
		}
		if ipc.Locate(in) != ipc.Locate(in) {
			t.Fatal("Locate is not deterministic")
		}
	})
}

// TestPropertyFrameRoundtrip (ipc/property/frame-roundtrip).
func TestPropertyFrameRoundtrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		req := ipc.Request{Op: rapid.String().Draw(t, "op"), Key: keyGen().Draw(t, "key")}
		var buf bytes.Buffer
		if err := ipc.WriteFrame(&buf, req); err != nil {
			t.Fatal(err)
		}
		var got ipc.Request
		if err := ipc.ReadFrame(&buf, &got); err != nil {
			t.Fatal(err)
		}
		// JSON replaces invalid UTF-8; compare what survives the wire twice
		var again bytes.Buffer
		_ = ipc.WriteFrame(&again, got)
		var twice ipc.Request
		_ = ipc.ReadFrame(&again, &twice)
		if twice != got {
			t.Fatalf("round trip not stable: %+v -> %+v", got, twice)
		}
	})
}

func TestFrameTooLarge(t *testing.T) {
	big := snapshot.Snapshot{WorkDir: string(bytes.Repeat([]byte("x"), ipc.MaxFrame))}
	if err := ipc.WriteFrame(&bytes.Buffer{}, big); !errors.Is(err, ipc.ErrFrameTooLarge) {
		t.Errorf("WriteFrame(too large) = %v", err)
	}
	header := []byte{0x7f, 0xff, 0xff, 0xff}
	var v any
	if err := ipc.ReadFrame(bytes.NewReader(header), &v); !errors.Is(err, ipc.ErrFrameTooLarge) {
		t.Errorf("ReadFrame(announced 2 GiB) = %v", err)
	}
}

// TestPropertyVersionsOrder (ipc/property/versions-order).
func TestPropertyVersionsOrder(t *testing.T) {
	version := rapid.Custom(func(t *rapid.T) string {
		if rapid.IntRange(0, 9).Draw(t, "dev") == 0 {
			return rapid.SampledFrom([]string{"", "dev"}).Draw(t, "devname")
		}
		v := fmt.Sprintf("v%d.%d.%d", rapid.IntRange(0, 3).Draw(t, "maj"), rapid.IntRange(0, 3).Draw(t, "min"), rapid.IntRange(0, 3).Draw(t, "pat"))
		if rapid.Bool().Draw(t, "pre") {
			v += "-rc" + fmt.Sprint(rapid.IntRange(1, 3).Draw(t, "rc"))
		}
		return v
	})
	rapid.Check(t, func(t *rapid.T) {
		a, b, c := version.Draw(t, "a"), version.Draw(t, "b"), version.Draw(t, "c")
		if ipc.CompareVersions(a, b) != -ipc.CompareVersions(b, a) {
			t.Fatalf("not antisymmetric: %s %s", a, b)
		}
		if ipc.CompareVersions(a, b) <= 0 && ipc.CompareVersions(b, c) <= 0 && ipc.CompareVersions(a, c) > 0 {
			t.Fatalf("not transitive: %s %s %s", a, b, c)
		}
	})
}

func TestCompatible(t *testing.T) {
	for p, want := range map[string]bool{
		ipc.Protocol: true, "statusline.ipc/v1.3": true, "statusline.ipc/v2": false,
		"other/v1": false, "": false, "statusline.ipc": false,
	} {
		if got := ipc.Compatible(p); got != want {
			t.Errorf("Compatible(%q) = %v, want %v", p, got, want)
		}
	}
	if ipc.CompareVersions("", "v0.0.1") >= 0 || ipc.CompareVersions("v1.0.0", "v1.0.0-rc1") <= 0 {
		t.Error("a development build must be older than any release, a release newer than its candidate")
	}
}
