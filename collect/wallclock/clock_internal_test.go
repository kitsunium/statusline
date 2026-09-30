package wallclock

import (
	"context"
	"testing"
	"time"
)

func TestNowIsTheWallClock(t *testing.T) {
	before := time.Now()
	got, err := NewClock().Now(context.Background())
	if err != nil || got.Before(before) || got.After(time.Now()) {
		t.Errorf("Now() = %v, %v; want between %v and now", got, err, before)
	}
}
