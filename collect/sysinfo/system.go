package sysinfo

import (
	"context"

	"github.com/kitsunium/statusline/snapshot"
)

// system holds nothing.
type system struct{}

func newSystem() *System { return &System{} }

func (a *System) info(context.Context) (snapshot.System, error) {
	return snapshot.System{OS: detectOS(), IsDocker: isDocker()}, nil
}
