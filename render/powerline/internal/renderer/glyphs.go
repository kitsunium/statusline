// Package renderer provides status line rendering.
package renderer

import "os"

// glyphsEnv names the environment variable selecting the glyph set.
const glyphsEnv string = "STATUSLINE_GLYPHS"

// GlyphSet holds every symbol the quota pills draw.
//
// The default set comes from the Nerd Font private-use ranges, like the rest of
// the status line. A terminal without a Nerd Font shows those as tofu, and a
// symbol nobody can read is worse than no symbol at all, so a plain-text set
// stands ready for that case.
type GlyphSet struct {
	Reset     string
	Project   string
	Overrun   string
	Ahead     string
	Behind    string
	Level     string
	Fast      string
	EffortOn  string
	EffortOff string
	Separator string
	Divider   string
	Ctx       string
	Quota     string
	Cost      string
	Health    string
	Worktree  string
	TaskOpen  string
	TaskDone  string
	Subagents string
	MCP       string
}

// nerdGlyphs is the default set, drawn from the Nerd Font ranges.
var nerdGlyphs = GlyphSet{
	Reset:     " ",
	Project:   " ",
	Overrun:   " ",
	Ahead:     "",
	Behind:    "",
	Level:     "",
	Fast:      "",
	EffortOn:  "●",
	EffortOff: "●",
	Separator: "│",
	Divider:   SepThinRight,
	Ctx:       "\uf1c0",
	Quota:     "",
	Cost:      "\uf155",
	Health:    "\U000F0674",
	Worktree:  "\uf126",
	TaskOpen:  "\u25a1",
	TaskDone:  "\u25a0",
	Subagents: "\U000F06A9",
	MCP:       "\U000F048D",
}

// textGlyphs is the fallback set: nothing outside printable ASCII, so it
// renders in any font. "T-" reads as a countdown without needing a clock.
var textGlyphs = GlyphSet{
	Reset:     "T-",
	Project:   ">",
	Overrun:   "!",
	Ahead:     "v",
	Behind:    "^",
	Level:     "=",
	Fast:      "F",
	EffortOn:  "#",
	EffortOff: "-",
	Separator: "|",
	Divider:   "|",
	Ctx:       "",
	Quota:     "",
	Cost:      "$",
	Health:    "*",
	Worktree:  "wt",
	TaskOpen:  "-",
	TaskDone:  "#",
	Subagents: "agents",
	MCP:       "MCP",
}

// glyphs is the active set, resolved once at startup.
var glyphs = glyphsFromEnv()

// glyphsFromEnv resolves the glyph set from the environment.
//
// Returns:
//   - GlyphSet: configured set, the Nerd Font one when unset or unknown
func glyphsFromEnv() GlyphSet {
	// Fall back to printable ASCII only when explicitly asked
	switch os.Getenv(glyphsEnv) {
	// Plain text for terminals without a Nerd Font
	case "text", "ascii":
		return textGlyphs
	// Anything else keeps the Nerd Font set
	default:
		return nerdGlyphs
	}
}
