// Package renderer provides status line rendering.
package renderer

import "unicode/utf8"

// escapeByte opens every ANSI escape sequence.
const escapeByte byte = 0x1b

// runeRange is an inclusive range of code points.
type runeRange struct {
	lo, hi rune
}

// zeroWidth lists the code points a terminal draws in no cell: combining
// marks, zero-width spaces and joiners, variation selectors.
var zeroWidth = [...]runeRange{
	{0x0300, 0x036F}, {0x200B, 0x200F}, {0x2060, 0x2064},
	{0xFE00, 0xFE0F}, {0xFE20, 0xFE2F}, {0xE0100, 0xE01EF},
}

// wideRanges lists the East Asian wide and fullwidth blocks, and the emoji
// blocks terminals draw in two cells. The Nerd Font private-use glyphs are
// not here: Nerd Fonts draw them in one cell, as the host measures them.
var wideRanges = [...]runeRange{
	{0x1100, 0x115F}, {0x231A, 0x231B}, {0x2329, 0x232A}, {0x23E9, 0x23EC},
	{0x23F0, 0x23F0}, {0x23F3, 0x23F3}, {0x25FD, 0x25FE}, {0x2614, 0x2615},
	{0x2648, 0x2653}, {0x26A1, 0x26A1}, {0x26AA, 0x26AB}, {0x26BD, 0x26BE},
	{0x26C4, 0x26C5}, {0x26D4, 0x26D4}, {0x26EA, 0x26EA}, {0x26F2, 0x26F5},
	{0x26FA, 0x26FD}, {0x2705, 0x2705}, {0x270A, 0x270B}, {0x2728, 0x2728},
	{0x274C, 0x274C}, {0x2753, 0x2755}, {0x2757, 0x2757}, {0x2795, 0x2797},
	{0x27B0, 0x27B0}, {0x27BF, 0x27BF}, {0x2B1B, 0x2B1C}, {0x2B50, 0x2B50},
	{0x2E80, 0x303E}, {0x3041, 0x33FF}, {0x3400, 0x4DBF}, {0x4E00, 0x9FFF},
	{0xA000, 0xA4CF}, {0xAC00, 0xD7A3}, {0xF900, 0xFAFF}, {0xFE30, 0xFE4F},
	{0xFF00, 0xFF60}, {0xFFE0, 0xFFE6}, {0x1F300, 0x1F64F}, {0x1F680, 0x1F6FF},
	{0x1F900, 0x1F9FF}, {0x1FA70, 0x1FAFF}, {0x20000, 0x3FFFD},
}

// inRanges reports whether r falls in one of the ranges.
//
// Params:
//   - r: code point to look up
//   - ranges: sorted inclusive ranges
//
// Returns:
//   - bool: true when a range holds r
func inRanges(r rune, ranges []runeRange) bool {
	// The tables are short: a linear scan beats a search on this size
	for _, rg := range ranges {
		// Sorted ranges: past r, nothing further can hold it
		if r < rg.lo {
			return false
		}
		// Inside this range
		if r <= rg.hi {
			return true
		}
	}
	return false
}

// RuneWidth returns the cells a terminal draws a code point in.
//
// Params:
//   - r: code point
//
// Returns:
//   - int: 0 for control and combining code points, 2 for wide ones, else 1
func RuneWidth(r rune) int {
	// ASCII is the bulk of the line: settle it first
	if r < 0x7F {
		// Control characters draw nothing
		if r < 0x20 {
			return 0
		}
		return 1
	}
	// C1 controls and the zero-width marks draw nothing
	if r < 0xA0 || inRanges(r, zeroWidth[:]) {
		return 0
	}
	// East Asian wide blocks and the emoji take two cells
	if r >= 0x1100 && inRanges(r, wideRanges[:]) {
		return 2
	}
	return 1
}

// VisibleWidth measures the cells a rendered string takes on screen.
//
// ANSI escape sequences are skipped: CSI sequences up to their final byte
// (0x40-0x7E), OSC sequences up to BEL or ST, and any other two-byte escape.
//
// Params:
//   - s: rendered string
//
// Returns:
//   - int: cells taken on one line
func VisibleWidth(s string) int {
	width := 0
	// Walk bytes, jumping over escapes, decoding the rest as runes
	for i := 0; i < len(s); {
		// An escape draws nothing: skip to its end
		if s[i] == escapeByte {
			i = skipEscape(s, i)
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		width += RuneWidth(r)
		i += size
	}
	return width
}

// skipEscape returns the index just past the escape sequence at i.
//
// Params:
//   - s: string holding the sequence
//   - i: index of the ESC byte
//
// Returns:
//   - int: index of the first byte after the sequence
func skipEscape(s string, i int) int {
	// A lone trailing ESC ends the string
	if i+1 >= len(s) {
		return len(s)
	}
	switch s[i+1] {
	// CSI: parameters and intermediates, then one final byte
	case '[':
		j := i + 2
		// Parameter and intermediate bytes lie below the final range
		for j < len(s) && (s[j] < 0x40 || s[j] > 0x7E) {
			j++
		}
		return min(j+1, len(s))
	// OSC: runs to BEL or to ESC-backslash
	case ']':
		j := i + 2
		// Look for either terminator
		for j < len(s) {
			// BEL ends the sequence
			if s[j] == 0x07 {
				return j + 1
			}
			// ST ends the sequence
			if s[j] == escapeByte && j+1 < len(s) && s[j+1] == '\\' {
				return j + 2
			}
			j++
		}
		return len(s)
	// Any other escape is two bytes long
	default:
		return i + 2
	}
}
