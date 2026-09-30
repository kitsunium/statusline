// Written once by kit gen, never again: this file is yours. The shells in
// design_gen.go delegate to what it holds; kit check reads a body that still
// answers errNotImplemented as a stub.

package gitcli

import (
	"context"

	"github.com/kitsunium/statusline/snapshot"
)

// git holds nothing: every call runs git afresh.
type git struct{}

func newGit() *Git { return &Git{} }

// status: outside a repository, the zero status.
func (a *Git) status(_ context.Context, dir string) (snapshot.GitStatus, error) {
	return readStatus(dir), nil
}

// diffStats: lines added and removed against HEAD.
func (a *Git) diffStats(_ context.Context, dir string) (snapshot.Changes, error) {
	return readDiffStats(dir), nil
}
