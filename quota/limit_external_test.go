package quota_test

import (
	"testing"
	"time"

	"github.com/kitsunium/statusline/quota"
)

func newWeekly(percent int, resetsAt time.Time) quota.Limit {
	return quota.NewLimit(quota.KindWeekly, "weekly", percent, resetsAt, quota.WeeklyWindow, quota.SourceAPI)
}

func newSession(percent int, resetsAt time.Time) quota.Limit {
	return quota.NewLimit(quota.KindSession, "session", percent, resetsAt, quota.SessionWindow, quota.SourceStdin)
}

func TestNewLimit_ClampsPercent(t *testing.T) {
	tests := []struct {
		name        string
		percent     int
		wantPercent int
	}{
		{name: "normal value", percent: 50, wantPercent: 50},
		{name: "zero", percent: 0, wantPercent: 0},
		{name: "max", percent: 100, wantPercent: 100},
		{name: "over max capped", percent: 150, wantPercent: 100},
		{name: "negative capped", percent: -10, wantPercent: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := newWeekly(tt.percent, time.Now())
			if l.Percent != tt.wantPercent {
				t.Errorf("NewLimit() Percent = %d, want %d", l.Percent, tt.wantPercent)
			}
		})
	}
}

func TestLimit_CursorPosition_Weekly(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		resetsAt time.Time
		wantMin  int
		wantMax  int
	}{
		{name: "reset passed", resetsAt: now.Add(-time.Hour), wantMin: 100, wantMax: 100},
		{name: "reset far future", resetsAt: now.Add(8 * 24 * time.Hour), wantMin: 0, wantMax: 0},
		{name: "half week remaining", resetsAt: now.Add(84 * time.Hour), wantMin: 45, wantMax: 55},
		{name: "one day remaining", resetsAt: now.Add(24 * time.Hour), wantMin: 80, wantMax: 90},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos := newWeekly(50, tt.resetsAt).CursorPosition(time.Now())
			if pos < tt.wantMin || pos > tt.wantMax {
				t.Errorf("CursorPosition() = %d, want between %d and %d", pos, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestLimit_CursorPosition_Session(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		resetsAt time.Time
		wantMin  int
		wantMax  int
	}{
		{name: "reset passed", resetsAt: now.Add(-time.Hour), wantMin: 100, wantMax: 100},
		{name: "reset far future", resetsAt: now.Add(6 * time.Hour), wantMin: 0, wantMax: 0},
		{name: "half session remaining", resetsAt: now.Add(150 * time.Minute), wantMin: 45, wantMax: 55},
		{name: "one hour remaining", resetsAt: now.Add(time.Hour), wantMin: 78, wantMax: 82},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos := newSession(40, tt.resetsAt).CursorPosition(time.Now())
			if pos < tt.wantMin || pos > tt.wantMax {
				t.Errorf("CursorPosition() = %d, want between %d and %d", pos, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestLimit_CursorPosition_ZeroWindow(t *testing.T) {
	l := quota.Limit{Percent: 50, ResetsAt: time.Now().Add(time.Hour), Source: quota.SourceAPI}
	if pos := l.CursorPosition(time.Now()); pos != 0 {
		t.Errorf("CursorPosition() with zero Window = %d, want 0", pos)
	}
}

func TestLimit_IsOnTrack(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name    string
		percent int
		want    bool
	}{
		{name: "on track (usage below cursor)", percent: 20, want: true},
		{name: "not on track (usage above cursor)", percent: 80, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newWeekly(tt.percent, now.Add(84*time.Hour)).IsOnTrack(time.Now()); got != tt.want {
				t.Errorf("IsOnTrack() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLimit_Pace(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name    string
		percent int
		wantMin int
		wantMax int
	}{
		{name: "ahead of the clock", percent: 20, wantMin: -35, wantMax: -25},
		{name: "level with the clock", percent: 50, wantMin: -5, wantMax: 5},
		{name: "behind the clock", percent: 80, wantMin: 25, wantMax: 35},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newWeekly(tt.percent, now.Add(84*time.Hour)).Pace(time.Now())
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("Pace() = %d, want between %d and %d", got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestLimit_Projected(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		percent  int
		resetsAt time.Time
		wantMin  int
		wantMax  int
	}{
		{name: "half window at 25 percent projects to 50", percent: 25, resetsAt: now.Add(84 * time.Hour), wantMin: 45, wantMax: 55},
		{name: "half window at 60 percent overruns", percent: 60, resetsAt: now.Add(84 * time.Hour), wantMin: 115, wantMax: 125},
		{name: "window not started yields zero", percent: 10, resetsAt: now.Add(8 * 24 * time.Hour), wantMin: 0, wantMax: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newWeekly(tt.percent, tt.resetsAt).Projected(time.Now())
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("Projected() = %d, want between %d and %d", got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestLimit_ExhaustsIn(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name        string
		percent     int
		resetsAt    time.Time
		wantExhaust bool
	}{
		{name: "sustainable burn never exhausts", percent: 25, resetsAt: now.Add(84 * time.Hour), wantExhaust: false},
		{name: "overrunning burn exhausts early", percent: 80, resetsAt: now.Add(84 * time.Hour), wantExhaust: true},
		{name: "no consumption never exhausts", percent: 0, resetsAt: now.Add(84 * time.Hour), wantExhaust: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			left, exhausts := newWeekly(tt.percent, tt.resetsAt).ExhaustsIn(time.Now())
			if exhausts != tt.wantExhaust {
				t.Errorf("ExhaustsIn() exhausts = %v, want %v", exhausts, tt.wantExhaust)
			}
			if exhausts && left <= 0 {
				t.Errorf("ExhaustsIn() returned non-positive duration %v", left)
			}
		})
	}
}

func TestLimit_Progress(t *testing.T) {
	tests := []struct {
		name        string
		percent     int
		wantPercent int
	}{
		{name: "50 percent", percent: 50, wantPercent: 50},
		{name: "zero", percent: 0, wantPercent: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newWeekly(tt.percent, time.Now()).Progress().Percent; got != tt.wantPercent {
				t.Errorf("Progress() Percent = %d, want %d", got, tt.wantPercent)
			}
		})
	}
}

func TestLimit_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		limit quota.Limit
		want  bool
	}{
		{name: "valid with reset time", limit: newWeekly(50, time.Now().Add(time.Hour)), want: true},
		{name: "invalid zero time", limit: newWeekly(50, time.Time{}), want: false},
		{name: "invalid without source", limit: quota.Limit{Percent: 50, ResetsAt: time.Now()}, want: false},
		{name: "context valid without reset time", limit: quota.NewLimit(quota.KindContext, "ctx", 10, time.Time{}, 0, quota.SourceStdin), want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.limit.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}
