// Package renderer provides status line rendering.
package renderer

import (
	"os"
	"strings"

	"github.com/kitsunium/statusline/render/powerline/internal/model"
)

// MCP pill constants.
const (
	// mcpLineEnv names the variable choosing the line of the MCP pill.
	mcpLineEnv string = "STATUSLINE_MCP_LINE"
	// mcpLineTwo keeps the pill on line two.
	mcpLineTwo string = "2"
	// mcpOffMark opens the count of disabled servers.
	mcpOffMark string = "\u00b7"
)

// mcpOnLine2 is true when the MCP servers are asked onto line two, as a
// pill of their own; by default they sit in the OS segment of line one.
// Resolved once at startup; tests replace it.
var mcpOnLine2 = os.Getenv(mcpLineEnv) == mcpLineTwo

// mcpSummary is everything the MCP pill says.
type mcpSummary struct {
	// on counts the enabled servers, undeclared ones being called included
	on int
	// off counts the disabled servers
	off int
	// busy is true while a call to any server is in flight
	busy bool
}

// summarizeMCP counts the servers and tells whether one is being called.
//
// Params:
//   - servers: servers to sum up
//
// Returns:
//   - mcpSummary: counts and call state
func summarizeMCP(servers model.MCPServers) mcpSummary {
	var s mcpSummary
	// One pass: count by state, note any call
	for _, srv := range servers {
		// Enabled and disabled are counted apart
		if srv.Enabled {
			s.on++
		} else {
			s.off++
		}
		s.busy = s.busy || srv.Busy
	}
	return s
}

// writeMCPInline sums the MCP servers up inside the OS segment, on its
// white ground: the MCP glyph in dark teal, the number of enabled servers
// in the OS ink, then, when some are disabled, a muted "·N" crossed out.
// While a call is in flight the glyph and the count become a chip, bold
// white on dark teal, taking exactly the cells they took at rest so the
// line does not shift. No server, nothing.
//
// Params:
//   - sb: string builder to write to
//   - s: servers summed up
func writeMCPInline(sb *strings.Builder, s mcpSummary) {
	// Nothing configured draws nothing
	if s.on+s.off == 0 {
		return
	}
	// Lit: one chip for glyph and count; at rest: each in its own ink
	if s.busy {
		sb.WriteString(BgMCPLabel + FgWhite + Bold + Labelled(glyphs.MCP, itoa(s.on)) + Reset)
	} else {
		glyph := glyphs.MCP
		// The text set spells the glyph out; it still needs its space
		sb.WriteString(BgWhite + FgMCPOnWhite + Bold + glyph + " " + Reset)
		sb.WriteString(BgWhite + FgBlack + Bold + itoa(s.on) + Reset)
	}
	// The disabled servers are a discreet suffix, never a label of their own
	if s.off > 0 {
		sb.WriteString(BgWhite + FgMCPMutedOnWhite + " " + mcpOffMark + StrikeMCP + itoa(s.off) + Reset)
	}
	sb.WriteString(BgWhite + " " + Reset)
}

// renderMCPPill renders the MCP servers as one small pill (line two, when
// STATUSLINE_MCP_LINE=2): the MCP glyph
// and the number of enabled servers (text glyphs: "MCP 7"), then, when some
// are disabled, a muted "·N" crossed out. While a call is in flight the
// whole pill takes the chip colours, bold white on dark teal; the disabled
// count stays, in pale teal. No server, no pill.
//
// Params:
//   - sb: string builder to write to
//   - servers: list of MCP servers
func (r *Powerline) renderMCPPill(sb *strings.Builder, servers model.MCPServers) {
	s := summarizeMCP(servers)
	// Nothing configured draws nothing
	if s.on+s.off == 0 {
		return
	}
	// At rest: dark ink on pale teal; lit: the chip colours
	bg, ink, cap, offInk := BgMCPEnabled, FgMCPEnabledText, FgMCPEnabled, FgMCPMuted
	if s.busy {
		bg, ink, cap, offInk = BgMCPLabel, FgWhite, FgMCPEnabledText, FgMCPEnabled
	}
	sb.WriteString(" " + cap + LeftRound + Reset)
	sb.WriteString(bg + ink + Bold + " " + Labelled(glyphs.MCP, itoa(s.on)) + Reset)
	// The disabled servers are a discreet suffix, never a label of their own
	if s.off > 0 {
		sb.WriteString(bg + offInk + " " + mcpOffMark + StrikeMCP + itoa(s.off) + Reset)
	}
	sb.WriteString(bg + " " + Reset + cap + RightRound + Reset)
}
