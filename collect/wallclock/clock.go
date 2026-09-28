package wallclock

import (
	"context"
	"time"
)

// clock is the wall clock; it holds nothing.
type clock struct{}

func newClock() *Clock { return &Clock{} }

func (a *Clock) now(context.Context) (time.Time, error) { return time.Now(), nil }
