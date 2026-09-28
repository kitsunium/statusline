package systemclock

import "time"

func (c *Clock) now() time.Time { return time.Now() }
