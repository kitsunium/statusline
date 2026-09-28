package credentials

import (
	"context"
	"os"
)

// credentials reads the process environment for CLAUDE_CONFIG_DIR.
type credentials struct {
	getenv func(string) string
}

func newCredentials() *Credentials { return &Credentials{credentials{getenv: os.Getenv}} }

// token is read on every call: never cached, refreshed or written.
func (a *Credentials) token(ctx context.Context) (string, error) {
	return readToken(ctx, a.getenv)
}
