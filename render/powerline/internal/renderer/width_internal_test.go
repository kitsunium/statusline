package renderer

import "testing"

func TestRuneWidth(t *testing.T) {
	tests := []struct {
		name string
		r    rune
		want int
	}{
		{"ascii", 'a', 1},
		{"control", '\t', 0},
		{"delete", 0x7F, 0},
		{"latin", 'é', 1},
		{"combining acute", '́', 0},
		{"zero width joiner", '‍', 0},
		{"variation selector", '️', 0},
		{"powerline arrow (PUA)", '', 1},
		{"nerd font glyph (PUA)", '', 1},
		{"nerd font glyph (supplementary PUA)", '\U000F0674', 1},
		{"box drawing", '━', 1},
		{"black circle", '●', 1},
		{"ellipsis", '…', 1},
		{"middle dot", '·', 1},
		{"cjk", '中', 2},
		{"hangul", '가', 2},
		{"fullwidth A", 'Ａ', 2},
		{"emoji", '\U0001F600', 2},
		{"rocket", '\U0001F680', 2},
		{"sparkles", '✨', 2},
	}
	for _, tt := range tests {
		if got := RuneWidth(tt.r); got != tt.want {
			t.Errorf("%s: RuneWidth(%U) = %d, want %d", tt.name, tt.r, got, tt.want)
		}
	}
}

func TestVisibleWidth(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want int
	}{
		{"empty", "", 0},
		{"plain", "abc", 3},
		{"sgr", "\033[1;38;5;23mab\033[0m", 2},
		{"truecolor", "\033[38;2;117;78;39mx\033[0m", 1},
		{"osc hyperlink BEL", "\033]8;;http://x\007link\033]8;;\007", 4},
		{"osc hyperlink ST", "\033]8;;http://x\033\\link\033]8;;\033\\", 4},
		{"two-byte escape", "\0337z", 1},
		{"lone trailing escape", "a\033", 1},
		{"unterminated csi", "a\033[38;5", 1},
		{"nerd glyphs", "  \U000F0674 ", 6},
		{"wide", "中文", 4},
		{"combining", "é", 1},
		{"invalid utf-8 counts one cell", "\xff", 1},
	}
	for _, tt := range tests {
		if got := VisibleWidth(tt.s); got != tt.want {
			t.Errorf("%s: VisibleWidth(%q) = %d, want %d", tt.name, tt.s, got, tt.want)
		}
	}
}

func TestWidthTablesAreSorted(t *testing.T) {
	for name, table := range map[string][]runeRange{"zeroWidth": zeroWidth[:], "wideRanges": wideRanges[:]} {
		for i, rg := range table {
			if rg.lo > rg.hi {
				t.Errorf("%s[%d]: %U > %U", name, i, rg.lo, rg.hi)
			}
			if i > 0 && table[i-1].hi >= rg.lo {
				t.Errorf("%s[%d]: overlaps or is out of order", name, i)
			}
		}
	}
}
