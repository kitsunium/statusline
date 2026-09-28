// Package sessions reads the host's session registry: one small file per
// running session, <config>/sessions/<pid>.json, naming its id and status.
//
// Exported API of design/domains/collect.yaml (collect/component/sessions).
// This file stands in for the shells kit generates (api_gen.go) until
// `kit gen` is available: every exported symbol only delegates to its
// unexported twin.
package sessions

import "github.com/kitsunium/statusline/collect/state"

// Registry implements collect/port/session-registry@v1.
type Registry struct {
	dir string
}

// New locates the registry from the environment.
func New() *Registry { return newRegistry() }

// Lookup returns the host process and whether it is busy on a turn; false
// when no entry names the session.
func (r *Registry) Lookup(sessionID string) (state.HostSession, bool) { return r.lookup(sessionID) }
