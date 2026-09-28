package renderer

import (
	"regexp"
	"strings"
	"testing"

	"github.com/kitsunium/statusline/render/powerline/internal/model"
)

// stripSGR removes the colour escapes, leaving what the terminal prints.
func stripSGR(s string) string {
	return regexp.MustCompile("\033\\[[0-9;]*m").ReplaceAllString(s, "")
}

// servers builds on enabled and off disabled servers, from mixed scopes.
func servers(on, off int) model.MCPServers {
	out := make(model.MCPServers, 0, on+off)
	scopes := []model.MCPSource{model.MCPSourceCLI, model.MCPSourceUser, model.MCPSourcePlugin}
	for i := range on {
		out = append(out, model.MCPServer{Name: "on" + itoa(i), Enabled: true, Source: scopes[i%len(scopes)]})
	}
	for i := range off {
		out = append(out, model.MCPServer{Name: "off" + itoa(i), Source: model.MCPSourceUser})
	}
	return out
}

func TestSummarizeMCP(t *testing.T) {
	list := servers(3, 2)
	if got := summarizeMCP(list); got != (mcpSummary{on: 3, off: 2}) {
		t.Errorf("summarizeMCP = %+v, want 3 on, 2 off, idle", got)
	}
	list[4].Busy = true
	if got := summarizeMCP(list); !got.busy {
		t.Error("a disabled server being called still lights the pill")
	}
	ghost := model.WithBusy(servers(1, 0), []string{"ghost"})
	if got := summarizeMCP(ghost); got != (mcpSummary{on: 2, busy: true}) {
		t.Errorf("an undeclared server being called counts as enabled, got %+v", got)
	}
}

func TestPowerline_renderMCPPill(t *testing.T) {
	head := " " + LeftRound + " " + glyphs.MCP + " "
	tests := []struct {
		name    string
		servers model.MCPServers
		want    string
	}{
		{name: "empty draws nothing", servers: model.MCPServers{}, want: ""},
		{name: "the total of enabled servers", servers: servers(7, 0), want: head + "7 " + RightRound},
		{name: "a discreet disabled suffix", servers: servers(7, 1), want: head + "7 ·1 " + RightRound},
		{name: "only disabled", servers: servers(0, 2), want: head + "0 ·2 " + RightRound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sb strings.Builder
			(&Powerline{}).renderMCPPill(&sb, tt.servers)
			if got := stripSGR(sb.String()); got != tt.want {
				t.Errorf("pill = %q, want %q", got, tt.want)
			}
		})
	}
}

// pillOf renders the line-two pill on its own.
func pillOf(list model.MCPServers) string {
	var sb strings.Builder
	(&Powerline{}).renderMCPPill(&sb, list)
	return sb.String()
}

func TestPowerline_renderMCPPillStyling(t *testing.T) {
	idle := pillOf(servers(2, 1))
	if !strings.HasPrefix(idle, " "+FgMCPEnabled+LeftRound+Reset+BgMCPEnabled+FgMCPEnabledText+Bold+" "+glyphs.MCP+" 2") {
		t.Errorf("at rest: bold dark ink on pale teal, got %q", idle)
	}
	if !strings.Contains(idle, BgMCPEnabled+FgMCPMuted+" "+mcpOffMark+StrikeMCP+"1"+Reset) {
		t.Errorf("at rest: the disabled count is muted and crossed out, got %q", idle)
	}
	list := servers(2, 1)
	list[0].Busy = true
	lit := pillOf(list)
	if !strings.HasPrefix(lit, " "+FgMCPEnabledText+LeftRound+Reset+BgMCPLabel+FgWhite+Bold+" "+glyphs.MCP+" 2") {
		t.Errorf("lit: the whole pill is bold white on dark teal, got %q", lit)
	}
	if !strings.Contains(lit, BgMCPLabel+FgMCPEnabled+" "+mcpOffMark+StrikeMCP+"1"+Reset) {
		t.Errorf("lit: the disabled count stays, pale teal on dark teal, got %q", lit)
	}
}

// inlineOf renders the OS-segment indicator on its own.
func inlineOf(list model.MCPServers) string {
	var sb strings.Builder
	writeMCPInline(&sb, summarizeMCP(list))
	return sb.String()
}

func TestWriteMCPInline(t *testing.T) {
	if got := inlineOf(nil); got != "" {
		t.Errorf("no server, nothing, got %q", got)
	}
	if got, want := stripSGR(inlineOf(servers(7, 0))), glyphs.MCP+" 7 "; got != want {
		t.Errorf("idle = %q, want %q", got, want)
	}
	if got, want := stripSGR(inlineOf(servers(7, 1))), glyphs.MCP+" 7 \u00b71 "; got != want {
		t.Errorf("with a disabled server = %q, want %q", got, want)
	}
	idle := inlineOf(servers(7, 2))
	for what, piece := range map[string]string{
		"the glyph in dark teal on white":       BgWhite + FgMCPOnWhite + Bold + glyphs.MCP + " " + Reset,
		"the count in the OS ink":               BgWhite + FgBlack + Bold + "7" + Reset,
		"the disabled count muted, crossed out": BgWhite + FgMCPMutedOnWhite + " " + mcpOffMark + StrikeMCP + "2" + Reset,
		"closed by a space on the white ground": BgWhite + " " + Reset,
	} {
		if !strings.Contains(idle, piece) {
			t.Errorf("idle: %s, got %q", what, idle)
		}
	}
	if strings.Contains(idle, LeftRound) || strings.Contains(idle, RightRound) {
		t.Errorf("inside the OS segment there are no caps, got %q", idle)
	}

	list := servers(7, 2)
	list[1].Busy = true
	lit := inlineOf(list)
	if !strings.HasPrefix(lit, BgMCPLabel+FgWhite+Bold+glyphs.MCP+" 7"+Reset) {
		t.Errorf("lit: glyph and count are one bold white chip on dark teal, got %q", lit)
	}
	if !strings.Contains(lit, BgWhite+FgMCPMutedOnWhite+" "+mcpOffMark+StrikeMCP+"2") {
		t.Errorf("lit: the disabled suffix stays, got %q", lit)
	}
	if VisibleWidth(lit) != VisibleWidth(idle) {
		t.Errorf("lighting up must not shift the line: %d vs %d cells", VisibleWidth(lit), VisibleWidth(idle))
	}
}

func TestWriteMCPInlineTextGlyphs(t *testing.T) {
	saved := glyphs
	t.Cleanup(func() { glyphs = saved })
	glyphs = textGlyphs
	if got := stripSGR(inlineOf(servers(7, 0))); got != "MCP 7 " {
		t.Errorf("text glyphs: want \"MCP 7 \", got %q", got)
	}
	list := servers(7, 0)
	list[0].Busy = true
	if got := stripSGR(inlineOf(list)); got != "MCP 7 " {
		t.Errorf("text glyphs, lit: want \"MCP 7 \", got %q", got)
	}
}

func TestPowerline_renderMCPPillTextGlyphs(t *testing.T) {
	saved := glyphs
	t.Cleanup(func() { glyphs = saved })
	glyphs = textGlyphs
	if got := stripSGR(pillOf(servers(7, 0))); !strings.Contains(got, " MCP 7 ") {
		t.Errorf("text glyphs: want \"MCP 7\", got %q", got)
	}
}

// withMCPLine sets the indicator's line for one test.
func withMCPLine(t *testing.T, line2 bool) {
	t.Helper()
	saved := mcpOnLine2
	t.Cleanup(func() { mcpOnLine2 = saved })
	mcpOnLine2 = line2
}

func TestOSSegmentOrder(t *testing.T) {
	var sb strings.Builder
	(&Powerline{}).renderOSSegment(&sb, model.SystemInfo{OS: model.OSLinux}, true, model.HealthOK, servers(7, 1), 2, BgBlue)
	got := stripSGR(sb.String())
	health, mcp, agents := strings.Index(got, glyphs.Health), strings.Index(got, glyphs.MCP+" 7"), strings.Index(got, glyphs.Subagents+" 2")
	if health < 0 || mcp < 0 || agents < 0 || !(health < mcp && mcp < agents) {
		t.Errorf("want OS icon, health, MCP, subagents; got %q", got)
	}
	if strings.Count(sb.String(), SepRight) != 1 {
		t.Errorf("the indicator stays inside the one OS segment, got %q", got)
	}
}

func TestMCPIndicatorInTheOSSegment(t *testing.T) {
	withMCPLine(t, false)
	for _, width := range []int{0, 200, 160, 120, 100, 80} {
		data := busyLine(width)
		data.MCP = servers(7, 1)
		out := (&Powerline{}).Render(data)
		line1, line2, _ := strings.Cut(out, "\n")
		os, _, _ := strings.Cut(line1, SepRight)
		if !strings.Contains(stripSGR(os), glyphs.MCP+" 7 \u00b71") {
			t.Errorf("COLUMNS=%d: the indicator sits in the OS segment, got %q", width, stripSGR(line1))
		}
		if strings.Contains(line2, glyphs.MCP) {
			t.Errorf("COLUMNS=%d: never on line two by default, got %q", width, stripSGR(line2))
		}
		if width > 0 && VisibleWidth(line1) > lineBudget(width) {
			t.Errorf("COLUMNS=%d: line one is %d wide, over %d", width, VisibleWidth(line1), lineBudget(width))
		}
	}
}

func TestMCPPillEnvKeepsItOnLineTwo(t *testing.T) {
	withMCPLine(t, true)
	data := busyLine(200)
	data.MCP = servers(7, 0)
	out := (&Powerline{}).Render(data)
	line1, line2, _ := strings.Cut(out, "\n")
	if strings.Contains(line1, glyphs.MCP) {
		t.Errorf("STATUSLINE_MCP_LINE=2: nothing in line one, got %q", stripSGR(line1))
	}
	if !strings.Contains(line2, FgMCPEnabled+LeftRound) || !strings.Contains(stripSGR(line2), glyphs.MCP+" 7") {
		t.Errorf("STATUSLINE_MCP_LINE=2: the pill is on line two, got %q", stripSGR(line2))
	}
}

func TestMCPIndicatorAbsentWithoutServers(t *testing.T) {
	withMCPLine(t, false)
	if out := (&Powerline{}).Render(busyLine(200)); strings.Contains(out, glyphs.MCP) {
		t.Errorf("no server, no indicator, got %q", stripSGR(out))
	}
}
