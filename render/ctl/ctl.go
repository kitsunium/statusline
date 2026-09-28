package ctl

import "context"

// execute: no daemon is an answer, not an error.
func (c *Status) execute(ctx context.Context, _ StatusInput) (StatusOutput, error) {
	st, err := c.deps.Daemon.Status(ctx)
	if err != nil {
		return StatusOutput{}, nil
	}
	return StatusOutput{Running: true, Status: st}, nil
}

// execute: stopping a daemon that is not running is not an error either.
func (c *Stop) execute(ctx context.Context, _ StopInput) (StopOutput, error) {
	if err := c.deps.Daemon.Stop(ctx); err != nil {
		return StopOutput{}, nil
	}
	return StopOutput{WasRunning: true}, nil
}
