package renderer

import "testing"

func TestTrueColorFromEnv(t *testing.T) {
	tests := []struct {
		name      string
		override  string
		colorterm string
		want      bool
	}{
		{name: "terminal announces truecolor", colorterm: "truecolor", want: true},
		{name: "terminal announces 24bit", colorterm: "24bit", want: true},
		{name: "nothing announced", want: false},
		{name: "unknown announcement", colorterm: "yes", want: false},
		{name: "forced 256 beats the terminal", override: "256", colorterm: "truecolor", want: false},
		{name: "forced truecolor beats silence", override: "truecolor", want: true},
		{name: "unknown override falls back to detection", override: "lots", colorterm: "truecolor", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(colorsEnv, tt.override)
			t.Setenv(colorTermEnv, tt.colorterm)
			if got := trueColorFromEnv(); got != tt.want {
				t.Errorf("trueColorFromEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPickInk(t *testing.T) {
	saved := trueColor
	t.Cleanup(func() { trueColor = saved })

	trueColor = true
	if got := pickInk(inkOpusTrue, inkOpus256); got != inkOpusTrue {
		t.Errorf("pickInk() with truecolor = %q, want the 24-bit ink", got)
	}
	trueColor = false
	if got := pickInk(inkOpusTrue, inkOpus256); got != inkOpus256 {
		t.Errorf("pickInk() without truecolor = %q, want the 256-colour ink", got)
	}
}
