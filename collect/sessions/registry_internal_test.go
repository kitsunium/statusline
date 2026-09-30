package sessions

import (
	"os"
	"path/filepath"
	"testing"
)

func writeEntry(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestNewLocatesTheRegistry(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv(configDirEnv, cfg)
	if got, want := newSessions().dir, filepath.Join(cfg, "sessions"); got != want {
		t.Errorf("dir = %q, want %q", got, want)
	}
}

func TestLookupWorking(t *testing.T) {
	dir := t.TempDir()
	const id = "00000000-0000-4000-8000-00000000000a"
	working := func() bool {
		h, _ := lookup(dir, id)
		return h.Working
	}
	if working() {
		t.Error("an empty registry reads as working")
	}

	writeEntry(t, dir, "1.json", `{"pid":1,"sessionId":"other","status":"busy"}`)
	writeEntry(t, dir, "2.json", `{"pid":2,"sessionId":"`)
	writeEntry(t, dir, "3.json", `{"pid":3,"sessionId":"other","status":"busy","note":"`+id+`"}`)
	writeEntry(t, dir, "4.key", id)
	if _, ok := lookup(dir, id); ok {
		t.Error("no entry is this session's, yet one was found")
	}

	writeEntry(t, dir, "5.json", `{"pid":5,"sessionId":"`+id+`","status":"idle"}`)
	if working() {
		t.Error("an idle session reads as working")
	}
	writeEntry(t, dir, "5.json", `{"pid":5,"sessionId":"`+id+`","status":"busy"}`)
	if !working() {
		t.Error("a busy session reads as idle")
	}
	writeEntry(t, dir, "5.json", `{"pid":5,"sessionId":"`+id+`","status":"waiting"}`)
	if working() {
		t.Error("a session waiting on the user reads as working")
	}
	if _, ok := lookup("", id); ok {
		t.Error("a registry without a directory found an entry")
	}
	if _, ok := lookup(dir, ""); ok {
		t.Error("an empty session id found an entry")
	}
}

func TestLookupPID(t *testing.T) {
	dir := t.TempDir()
	const id = "00000000-0000-4000-8000-00000000000b"
	writeEntry(t, dir, "1.json", `{"pid":1,"sessionId":"other","status":"busy"}`)
	writeEntry(t, dir, "5752.json", `{"pid":5752,"sessionId":"`+id+`","status":"busy"}`)
	if h, ok := lookup(dir, id); !ok || h.PID != 5752 {
		t.Errorf("Lookup() = %+v, %v; want pid 5752", h, ok)
	}
	// The daemon rescans on every collection: a change is seen at once
	writeEntry(t, dir, "5752.json", `{"pid":5752,"sessionId":"`+id+`","status":"idle"}`)
	if h, _ := lookup(dir, id); h.Working {
		t.Error("Lookup() did not see the session go idle")
	}
	writeEntry(t, dir, "9.json", `{"pid":-3,"sessionId":"neg","status":"busy"}`)
	if h, _ := lookup(dir, "neg"); h.PID != 0 {
		t.Errorf("PID of a negative pid = %d, want 0", h.PID)
	}
}
