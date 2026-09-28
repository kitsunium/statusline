package renderer

import "testing"

func TestFilledCells(t *testing.T) {
	tests := []struct {
		name    string
		percent int
		width   int
		want    int
	}{
		{name: "nothing spent shows nothing", percent: 0, width: 10, want: 0},
		{name: "one percent still shows", percent: 1, width: 10, want: 1},
		{name: "nine percent shows", percent: 9, width: 10, want: 1},
		{name: "ten percent is one cell either way", percent: 10, width: 10, want: 1},
		{name: "half", percent: 50, width: 10, want: 5},
		{name: "ninety-nine is not full", percent: 99, width: 10, want: 9},
		{name: "full", percent: 100, width: 10, want: 10},
		{name: "narrow bar, low percentage", percent: 3, width: 4, want: 1},
		{name: "wide bar, low percentage", percent: 1, width: 20, want: 1},
		{name: "wide bar, full", percent: 100, width: 20, want: 20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filledCells(tt.percent, tt.width); got != tt.want {
				t.Errorf("filledCells(%d, %d) = %d, want %d", tt.percent, tt.width, got, tt.want)
			}
		})
	}
}

func TestFilledCells_OnlyEmptyAtZeroOnlyFullAtHundred(t *testing.T) {
	// These are the two readings a bar has to get right: an untouched quota and
	// a spent one. Anything in between must look like neither.
	for _, width := range []int{4, 10, 12, 20} {
		for p := 1; p < 100; p++ {
			got := filledCells(p, width)
			if got == 0 {
				t.Errorf("filledCells(%d, %d) reads as untouched", p, width)
			}
			if got >= width {
				t.Errorf("filledCells(%d, %d) reads as full", p, width)
			}
		}
		if filledCells(0, width) != 0 {
			t.Errorf("filledCells(0, %d) should be empty", width)
		}
		if filledCells(100, width) != width {
			t.Errorf("filledCells(100, %d) should be full", width)
		}
	}
}

func TestFilledCells_NeverDecreases(t *testing.T) {
	// A bar that shrinks as consumption rises would be worse than no bar
	prev := 0
	for p := 0; p <= 100; p++ {
		got := filledCells(p, 10)
		if got < prev {
			t.Fatalf("filledCells(%d, 10) = %d, less than the previous %d", p, got, prev)
		}
		prev = got
	}
}
