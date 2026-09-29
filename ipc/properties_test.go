// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package ipc

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"pgregory.net/rapid"

	"github.com/kitsunium/statusline/snapshot"
)

func keyGen() *rapid.Generator[Key] {
	return rapid.Custom(func(t *rapid.T) Key {
		return Key{
			SessionID:      rapid.String().Draw(t, "session"),
			TranscriptPath: rapid.String().Draw(t, "transcript"),
			SessionDir:     rapid.String().Draw(t, "dir"),
			TaskListID:     rapid.String().Draw(t, "list"),
		}
	})
}

// propertyKeyHashStable (ipc/property/key-hash-stable).
func propertyKeyHashStable(t *testing.T, seed uint64) {
	t.Logf("design seed %d (rapid v1.2 takes its own seed from -rapid.seed)", seed)
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

// propertyLocateDistinct (ipc/property/locate-distinct).
func propertyLocateDistinct(t *testing.T, seed uint64) {
	t.Logf("design seed %d (rapid v1.2 takes its own seed from -rapid.seed)", seed)
	rapid.Check(t, func(t *rapid.T) {
		in := LocateInput{RuntimeDir: "/run/user/1000/daemon", Key: rapid.StringMatching(`[0-9a-f]{8,16}`).Draw(t, "key")}
		other := in
		other.Key += rapid.StringMatching(`[0-9a-f]{1,4}`).Draw(t, "more")
		if Locate(in).Socket == Locate(other).Socket {
			t.Fatal("two instances share a socket")
		}
		if Locate(in) != Locate(in) {
			t.Fatal("Locate is not deterministic")
		}
	})
}

// propertyReadFrameRoundtrip (ipc/property/frame-roundtrip).
func propertyReadFrameRoundtrip(t *testing.T, seed uint64) {
	t.Logf("design seed %d (rapid v1.2 takes its own seed from -rapid.seed)", seed)
	rapid.Check(t, func(t *rapid.T) {
		req := Request{Op: rapid.String().Draw(t, "op"), Key: keyGen().Draw(t, "key")}
		var buf bytes.Buffer
		if err := WriteFrame(&buf, req); err != nil {
			t.Fatal(err)
		}
		var got Request
		if err := ReadFrame(&buf, &got); err != nil {
			t.Fatal(err)
		}
		// JSON replaces invalid UTF-8; compare what survives the wire twice
		var again bytes.Buffer
		_ = WriteFrame(&again, got)
		var twice Request
		_ = ReadFrame(&again, &twice)
		if twice != got {
			t.Fatalf("round trip not stable: %+v -> %+v", got, twice)
		}
	})
}

func TestFrameTooLarge(t *testing.T) {
	big := snapshot.Snapshot{WorkDir: string(bytes.Repeat([]byte("x"), MaxFrame))}
	if err := WriteFrame(&bytes.Buffer{}, big); !errors.Is(err, ErrFrameTooLarge) {
		t.Errorf("WriteFrame(too large) = %v", err)
	}
	header := []byte{0x7f, 0xff, 0xff, 0xff}
	var v any
	if err := ReadFrame(bytes.NewReader(header), &v); !errors.Is(err, ErrFrameTooLarge) {
		t.Errorf("ReadFrame(announced 2 GiB) = %v", err)
	}
}

// propertyCompareVersionsOrder (ipc/property/versions-order).
func propertyCompareVersionsOrder(t *testing.T, seed uint64) {
	t.Logf("design seed %d (rapid v1.2 takes its own seed from -rapid.seed)", seed)
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
		if CompareVersions(a, b) != -CompareVersions(b, a) {
			t.Fatalf("not antisymmetric: %s %s", a, b)
		}
		if CompareVersions(a, b) <= 0 && CompareVersions(b, c) <= 0 && CompareVersions(a, c) > 0 {
			t.Fatalf("not transitive: %s %s %s", a, b, c)
		}
	})
}

func TestCompatible(t *testing.T) {
	for p, want := range map[string]bool{
		Protocol: true, "statusline.ipc/v1.3": true, "statusline.ipc/v2": false,
		"other/v1": false, "": false, "statusline.ipc": false,
	} {
		if got := Compatible(p); got != want {
			t.Errorf("Compatible(%q) = %v, want %v", p, got, want)
		}
	}
	if CompareVersions("", "v0.0.1") >= 0 || CompareVersions("v1.0.0", "v1.0.0-rc1") <= 0 {
		t.Error("a development build must be older than any release, a release newer than its candidate")
	}
}
