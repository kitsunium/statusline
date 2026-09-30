package transcripts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTail(t *testing.T) {
	path := filepath.Join(t.TempDir(), "t.jsonl")
	if _, err := tail(path, 10); err == nil {
		t.Error("a missing file must fail")
	}
	if err := os.WriteFile(path, []byte("aaaa\nbbbb\ncc\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		size int64
		want string
	}{
		{size: 100, want: "aaaa\nbbbb\ncc\n"},
		{size: 6, want: "cc\n"},
		{size: 2, want: ""},
	}
	for _, tt := range tests {
		got, err := tail(path, tt.size)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != tt.want {
			t.Errorf("tail(%d) = %q, want %q", tt.size, got, tt.want)
		}
	}
	if err := os.WriteFile(path, []byte(strings.Repeat("x", 50)), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, _ := tail(path, 10); len(got) != 0 {
		t.Errorf("a tail inside one line = %q, want nothing", got)
	}
}
