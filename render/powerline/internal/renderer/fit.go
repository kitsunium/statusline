// Package renderer provides status line rendering.
package renderer

import (
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/kitsunium/statusline/render/powerline/internal/model"
)

// Adaptive line constants.
const (
	// lineMargin is the number of columns kept free at the right of line
	// one. The host pads the status line and draws its own notices beside
	// it; a line that ends exactly at the edge wraps on the slightest
	// disagreement about a glyph's width.
	lineMargin int = 4
	// passStride separates two passes in the order of the steps: level n
	// of a segment of weight w shrinks at (n-1)*passStride + w. A weight
	// above passStride holds a segment back to a later pass.
	passStride int = 100
	// maxWeight bounds a weight given in STATUSLINE_WEIGHTS.
	maxWeight int = 999
	// weightsEnv names the variable overriding the default weights.
	weightsEnv string = "STATUSLINE_WEIGHTS"
	// branchEllipsis ends a shortened branch name.
	branchEllipsis string = "\u2026"
)

// segID names a line-one segment that can shrink. The OS segment (OS
// icon, health, MCP, subagents) is not one: it never shrinks.
type segID int

// Shrinkable segments of line one.
const (
	segContext segID = iota
	segScoped
	segWeekly
	segSession
	segPath
	segBranch
	segChanges
	segModel
	segCount
)

// segPolicy is how one segment condenses: its levels, richest first, each
// meaningful on its own, and its weight — the lower, the sooner it shrinks.
type segPolicy struct {
	name   string
	weight int
	levels []string
}

// Path and branch budgets per level.
var (
	// pathBudgets is the path budget per level; 1 leaves ".../<last>",
	// the last level hides the path (only inside a repository).
	pathBudgets = [...]int{0, 20, 1, 0}
	// branchBudgets is the branch budget in runes per level, 0 = whole.
	branchBudgets = [...]int{0, 20, 12, 8}
)

// condensePolicy is the condensing policy of line one, one row per segment.
//
// The condenser works in passes: pass n lowers every segment that still has
// an n-th step by one level, lowest weight first, and stops as soon as the
// line fits. With the default weights the steps run in the order the user
// set: the bars (context, scoped, weekly, session), the path and branch to
// 20, the countdowns, the path to its last element and the branch to 12,
// the quota names to their initial; then changes (240), the hidden path,
// the model icon (255) and the branch to 8, held back to the third pass by
// their weight. STATUSLINE_WEIGHTS=name=weight,… overrides weights.
var condensePolicy = [segCount]segPolicy{
	segContext: {name: "context", weight: 10, levels: []string{"bar + label + %", "icon + %"}},
	segScoped:  {name: "scoped", weight: 20, levels: []string{"bar + label + % + countdown", "label + % + countdown", "label + %", "initial + %"}},
	segWeekly:  {name: "weekly", weight: 30, levels: []string{"bar + label + % + countdown", "label + % + countdown", "label + %", "initial + %"}},
	segSession: {name: "session", weight: 40, levels: []string{"bar + % + countdown", "% + countdown", "%"}},
	segPath:    {name: "path", weight: 50, levels: []string{"30 cells", "20 cells", "last element", "hidden (inside a repository)"}},
	segBranch:  {name: "branch", weight: 60, levels: []string{"whole", "20 runes", "12 runes", "8 runes"}},
	segChanges: {name: "changes", weight: 240, levels: []string{"shown", "hidden"}},
	segModel:   {name: "model", weight: 255, levels: []string{"icon + name", "name"}},
}

// lineFit is the level of every shrinkable segment; the zero value draws
// everything at its richest.
type lineFit [segCount]int

// fitStep lowers one segment to one level.
type fitStep struct {
	seg   segID
	level int
}

// Describe names the step for people: "context: icon + %".
//
// Returns:
//   - string: segment name and the level it reaches
func (s fitStep) Describe() string {
	pol := condensePolicy[s.seg]
	return pol.name + ": " + pol.levels[s.level]
}

// fitSteps and fitLevels are resolved once, from the policy and the
// weights in the environment.
var fitSteps, fitLevels = buildFitLevels(weightsFromEnv(os.Getenv(weightsEnv)))

// weightsFromEnv reads STATUSLINE_WEIGHTS over the default weights.
//
// Entries are "name=weight", comma-separated; a weight is an integer from
// 0 to 999. An unknown name, a malformed entry or a weight out of range is
// ignored, the rest applies.
//
// Params:
//   - value: raw variable value
//
// Returns:
//   - [segCount]int: weight per segment
func weightsFromEnv(value string) [segCount]int {
	var weights [segCount]int
	// Start from the policy
	for id, pol := range condensePolicy {
		weights[id] = pol.weight
	}
	// Apply each well-formed entry
	for _, entry := range strings.Split(value, ",") {
		name, raw, found := strings.Cut(strings.TrimSpace(entry), "=")
		weight, err := strconv.Atoi(raw)
		// A malformed entry changes nothing
		if !found || err != nil || weight < 0 || weight > maxWeight {
			continue
		}
		// Only a known segment takes a weight
		for id, pol := range condensePolicy {
			// Names are matched exactly
			if pol.name == name {
				weights[id] = weight
			}
		}
	}
	return weights
}

// buildFitLevels orders every step of every segment and accumulates them.
//
// Step n of a segment of weight w is ranked (n-1)*passStride + w: all the
// first steps come before the second ones, and so on, lowest weight first
// within a pass; ties keep the policy order.
//
// Params:
//   - weights: weight per segment
//
// Returns:
//   - []fitStep: steps in the order they are taken
//   - []lineFit: the full line, then the state after each step
func buildFitLevels(weights [segCount]int) ([]fitStep, []lineFit) {
	type ranked struct {
		step fitStep
		rank int
	}
	var all []ranked
	// One step per level below the richest
	for id, pol := range condensePolicy {
		for level := 1; level < len(pol.levels); level++ {
			all = append(all, ranked{step: fitStep{seg: segID(id), level: level}, rank: (level-1)*passStride + weights[id]})
		}
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].rank < all[j].rank })

	steps := make([]fitStep, 0, len(all))
	levels := make([]lineFit, 0, len(all)+1)
	var fit lineFit
	levels = append(levels, fit)
	// Each state is the previous one with one segment a level lower
	for _, r := range all {
		fit[r.step.seg] = r.step.level
		steps = append(steps, r.step)
		levels = append(levels, fit)
	}
	return steps, levels
}

// quotaSeg maps a quota kind onto its segment.
//
// Params:
//   - kind: quota kind
//
// Returns:
//   - segID: its segment
//   - bool: false for a kind that never shrinks
func quotaSeg(kind model.LimitKind) (segID, bool) {
	// Each quota kind is a segment of its own
	switch kind {
	// The conversation window
	case model.KindContext:
		return segContext, true
	// The quota scoped to the model in use
	case model.KindScoped:
		return segScoped, true
	// The plan-wide weekly quota
	case model.KindWeekly:
		return segWeekly, true
	// The session quota beside the model name
	case model.KindSession:
		return segSession, true
	// Any other kind is drawn whole
	default:
		return 0, false
	}
}

// dropBar reports whether a quota of this kind loses its bar.
//
// Params:
//   - kind: quota kind
//
// Returns:
//   - bool: true from the first level on
func (f lineFit) dropBar(kind model.LimitKind) bool {
	seg, ok := quotaSeg(kind)
	return ok && f[seg] >= 1
}

// dropCountdown reports whether a quota of this kind loses its countdown.
//
// Params:
//   - kind: quota kind
//
// Returns:
//   - bool: true from the second level on
func (f lineFit) dropCountdown(kind model.LimitKind) bool {
	seg, ok := quotaSeg(kind)
	return ok && seg != segContext && f[seg] >= 2
}

// shortName reports whether a quota of this kind is named by its initial.
//
// Params:
//   - kind: quota kind
//
// Returns:
//   - bool: true at the third level, for the weekly and scoped quotas
func (f lineFit) shortName(kind model.LimitKind) bool {
	seg, ok := quotaSeg(kind)
	return ok && (seg == segScoped || seg == segWeekly) && f[seg] >= 3
}

// pathMax returns the path budget, 0 for the default one.
//
// Returns:
//   - int: budget for TruncatePath
func (f lineFit) pathMax() int {
	return pathBudgets[f[segPath]]
}

// dropPath reports whether the path is hidden (inside a repository).
//
// Returns:
//   - bool: true at the last path level
func (f lineFit) dropPath() bool {
	return f[segPath] == len(pathBudgets)-1
}

// branchMax returns the branch budget in runes, 0 for the whole name.
//
// Returns:
//   - int: budget for TruncateBranch
func (f lineFit) branchMax() int {
	return branchBudgets[f[segBranch]]
}

// dropChanges reports whether the added/removed lines are hidden.
//
// Returns:
//   - bool: true from the first level on
func (f lineFit) dropChanges() bool {
	return f[segChanges] >= 1
}

// dropModelIcon reports whether the model loses its icon.
//
// Returns:
//   - bool: true from the first level on
func (f lineFit) dropModelIcon() bool {
	return f[segModel] >= 1
}

// presentSegments tells which shrinkable segments the data draws, so a
// report does not name a step that had nothing to shrink.
//
// Params:
//   - data: status line data
//
// Returns:
//   - [segCount]bool: true for each segment on show
func presentSegments(data model.StatusLineData) [segCount]bool {
	var present [segCount]bool
	// Quotas: the chained context and the ones inside the model segment
	for _, seg := range quotaSegments(data) {
		present[segContext] = present[segContext] || seg.limit.Kind == model.KindContext
	}
	for _, q := range modelQuotas(data) {
		// Mark the segment each quota belongs to
		if seg, ok := quotaSeg(q.Kind); ok {
			present[seg] = true
		}
	}
	present[segPath] = data.Dir != ""
	present[segBranch] = data.Git.IsInRepo()
	present[segChanges] = data.Changes.HasChanges()
	present[segModel] = data.Icons.Model
	return present
}

// Condensed describes where line one stands after fitting the data:
// "context: icon + %", one entry per segment that shrank, in policy order.
//
// Params:
//   - data: status line data, with the terminal width
//
// Returns:
//   - []string: shrunken segments and their level, empty for a full line
func Condensed(data model.StatusLineData) []string {
	_, level, _ := fitLine1(lineBudget(data.Terminal.Width), func(sb *strings.Builder, fit lineFit) {
		(&Powerline{}).renderLine1Fit(sb, data, fit)
	})
	fit := fitLevels[level]
	present := presentSegments(data)
	var out []string
	// Name every segment on show below its richest level
	for id, lvl := range fit {
		// A segment at its richest, or not drawn at all, says nothing
		if lvl == 0 || !present[id] {
			continue
		}
		out = append(out, fitStep{seg: segID(id), level: lvl}.Describe())
	}
	return out
}

// lineBudget returns the columns line one may take.
//
// Params:
//   - width: terminal width, 0 or less when unknown
//
// Returns:
//   - int: budget in cells, 0 when nothing is to be fitted
func lineBudget(width int) int {
	// An unknown width is no constraint
	if width <= 0 {
		return 0
	}
	return max(width-lineMargin, 1)
}

// fitLine1 renders line one at the first level that fits the budget.
//
// Every state only gives up more than the one before it, so the widths
// never grow along them; the result is the one a step-by-step walk that
// re-measures after each step would reach, and the first level that fits is found by
// bisection: the full line first (the usual case, one render), then at
// most four more renders over the remaining levels.
//
// Params:
//   - budget: cells line one may take, 0 for no limit
//   - render: draws line one with the given fit
//
// Returns:
//   - string: the rendered line
//   - int: the level chosen, len(fitLevels)-1 when even the last overflows
//   - bool: false when even the last level overflows
func fitLine1(budget int, render func(*strings.Builder, lineFit)) (string, int, bool) {
	draw := func(level int) string {
		var sb strings.Builder
		render(&sb, fitLevels[level])
		return sb.String()
	}
	full := draw(0)
	// No budget, or a full line that fits, needs no search
	if budget <= 0 || VisibleWidth(full) <= budget {
		return full, 0, true
	}
	last := len(fitLevels) - 1
	best, bestLine := last, ""
	lo, hi := 1, last
	// Narrow down to the first level that fits
	for lo <= hi {
		mid := (lo + hi) / 2
		line := draw(mid)
		// A fit moves the search left, an overflow right
		if VisibleWidth(line) <= budget {
			best, bestLine, hi = mid, line, mid-1
		} else {
			lo = mid + 1
		}
	}
	// Nothing fits: keep the tightest line, the host wraps what is left
	if bestLine == "" {
		return draw(last), last, false
	}
	return bestLine, best, true
}

// quotaLabel names a quota inside the model segment.
//
// Params:
//   - limit: quota to name
//
// Returns:
//   - string: full label, or its initial once names have to give way
func (f lineFit) quotaLabel(limit model.Limit) string {
	// The full label reads best whenever there is room for it
	if !f.shortName(limit.Kind) {
		return QuotaLabel(limit)
	}
	name := nameWeekly
	// A scoped quota is named by its model family
	if limit.Kind == model.KindScoped {
		name = Capitalise(limit.Label)
	}
	// Nothing to shorten: keep whatever label it has
	if limit.Kind != model.KindWeekly && limit.Kind != model.KindScoped || name == "" {
		return QuotaLabel(limit)
	}
	return Labelled(glyphs.Quota, string([]rune(name)[:1]))
}

// TruncateBranch shortens a branch name to max runes, keeping its start.
//
// Params:
//   - branch: branch name
//   - maxRunes: budget in runes, 0 or less for no limit
//
// Returns:
//   - string: the name, or its start followed by an ellipsis
func TruncateBranch(branch string, maxRunes int) string {
	runes := []rune(branch)
	// No budget, or a name that already fits, stays whole
	if maxRunes <= 0 || len(runes) <= maxRunes {
		return branch
	}
	// The ellipsis takes one of the runes
	if maxRunes == 1 {
		return branchEllipsis
	}
	return string(runes[:maxRunes-1]) + branchEllipsis
}
