package credentials

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestTokenIsReadOnEveryCall(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("the keychain comes first on macOS")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	s := &Credentials{credentials{getenv: func(string) string { return "" }}}
	if _, err := s.Token(context.Background()); err == nil {
		t.Error("a token was found where there is none")
	}
	path := filepath.Join(home, ".claude", credentialsFileName)
	write(t, path, `{"claudeAiOauth":{"accessToken":" first "}}`)
	if got, _ := s.Token(context.Background()); got != "first" {
		t.Errorf("Token() = %q, want first", got)
	}
	// A rotated token is seen at once: nothing is cached
	write(t, path, `{"claudeAiOauth":{"accessToken":"second"}}`)
	if got, _ := s.Token(context.Background()); got != "second" {
		t.Errorf("Token() after rotation = %q, want second", got)
	}
	write(t, path, `{broken`)
	if _, err := s.Token(context.Background()); err == nil {
		t.Error("a malformed file yielded a token")
	}
}

func TestTokenRelocatedConfigFirst(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("the keychain comes first on macOS")
	}
	home, cfg := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	write(t, filepath.Join(home, ".claude", credentialsFileName), `{"claudeAiOauth":{"accessToken":"home"}}`)
	write(t, filepath.Join(cfg, credentialsFileName), `{"claudeAiOauth":{"accessToken":"relocated"}}`)
	s := &Credentials{credentials{getenv: func(key string) string {
		if key == configDirEnv {
			return cfg
		}
		return ""
	}}}
	if got, _ := s.Token(context.Background()); got != "relocated" {
		t.Errorf("Token() = %q, want the relocated one", got)
	}
}
