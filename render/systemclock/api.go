// Package systemclock is the client's wall clock.
//
// Exported API of design/domains/render.yaml (render/component/systemclock).
// This file stands in for the shells kit generates (api_gen.go) until
// `kit gen` is available.
package systemclock

import "time"

// Clock implements render/port/clock@v1.
type Clock struct{}

// New returns the wall clock.
func New() *Clock { return &Clock{} }

// Now returns the current instant.
func (c *Clock) Now() time.Time { return c.now() }
