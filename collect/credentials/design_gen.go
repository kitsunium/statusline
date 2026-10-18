// Code generated from design/v1; DO NOT EDIT.
// design:digest sha256:f7116793c8d8ac76611661ce6107f6662b6eb477309df7bb89556666304dc4e6

// Package credentials is the adapter component credentials of domain collect.
//
// Its structure is generated from design/ (ADR 0010); its bodies are the
// hand-written twins beside this file.
package credentials

import (
	"context"

	"github.com/kitsunium/sdk/pkg/v1/errs"
	"github.com/kitsunium/statusline/collect/collector"
)

// errNotImplemented is what a skeleton answers until its body is written:
// a twin that still returns it is a stub.
var errNotImplemented = errs.New(errs.Pack(64, 3, 11, 255), "NOT_IMPLEMENTED", "not implemented yet", "credentials: a hand-written body is still a skeleton")

// Credentials is an adapter of the design.
type Credentials struct {
	credentials // its own state, hand-written
}

// NewCredentials builds Credentials: its body is the hand-written newCredentials.
func NewCredentials() *Credentials {
	return newCredentials()
}

// Token is collect/port/TokenStore@v1's.
func (a *Credentials) Token(ctx context.Context) (string, error) {
	return a.token(ctx)
}

var _ collector.TokenStoreV1 = (*Credentials)(nil)
