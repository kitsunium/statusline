// Package sysinfo reports the operating system and whether it is a
// container.
//
// Exported API of design/domains/collect.yaml (collect/component/sysinfo).
// This file stands in for the shells kit generates (api_gen.go) until
// `kit gen` is available: every exported symbol only delegates to its
// unexported twin.
package sysinfo

import "github.com/kitsunium/statusline/snapshot"

// Reader implements collect/port/system-info@v1.
type Reader struct{}

// New returns a system reader.
func New() *Reader { return &Reader{} }

// Info returns the operating system and the container flag.
func (r *Reader) Info() snapshot.System { return r.info() }
