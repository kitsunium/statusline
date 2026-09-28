// Package credentials reads the host's OAuth token, on every call: never
// cached, never refreshed, never written.
//
// Exported API of design/domains/collect.yaml (collect/component/credentials).
// This file stands in for the shells kit generates (api_gen.go) until
// `kit gen` is available: every exported symbol only delegates to its
// unexported twin.
package credentials

import "context"

// Store implements collect/port/token-store@v1.
type Store struct {
	getenv func(string) string
}

// New returns a token reader over the given environment.
func New(getenv func(string) string) *Store { return &Store{getenv: getenv} }

// Token returns the current access token: the macOS keychain first, then
// <config>/.credentials.json, then ~/.claude/.credentials.json.
func (s *Store) Token(ctx context.Context) (string, error) { return s.token(ctx) }
