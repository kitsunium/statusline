// Package gitcli reads the repository state through the git CLI.
//
// Exported API of design/domains/collect.yaml (collect/component/gitcli).
// This file stands in for the shells kit generates (api_gen.go) until
// `kit gen` is available: every exported symbol only delegates to its
// unexported twin.
package gitcli

import "github.com/kitsunium/statusline/snapshot"

// Repository implements collect/port/repository@v1 with the git CLI.
type Repository struct{}

// New returns a repository reader.
func New() *Repository { return newRepository() }

// Status returns the branch, the change counts and the linked worktrees of
// the work tree holding dir; the zero status outside a repository.
func (r *Repository) Status(dir string) snapshot.GitStatus { return r.status(dir) }

// DiffStats returns the lines added and removed against HEAD.
func (r *Repository) DiffStats(dir string) snapshot.Changes { return r.diffStats(dir) }
