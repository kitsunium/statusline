# ADR 0002 — Line one condenses by levels and weights

- **Status**: Accepted (carried over from kodflow/status-line, frozen at 207dfe2)
- **Date**: 2026-09-28

## Decision

Line one fits `COLUMNS − 4` (the host pads the line; a line exactly at the
edge wraps on the slightest disagreement about a glyph's width). Each segment
has levels from richest to leanest, each legible alone, and a weight: the
lower, the sooner it gives way. Pass n lowers by one level every segment that
still has one, by increasing weight; the step n of a segment of weight w ranks
(n−1)·100 + w, so a weight above 100 holds a segment back for a later pass.
`STATUSLINE_WEIGHTS` overrides weights (0–999, malformed entries ignored).
The OS segment (with the MCP indicator) never shrinks; line two is never
shortened. The successive states are cumulative and decreasing in width, so
the first that fits is found by bisection. The oracle (`testdata/parity`)
pins the result from 40 to 200 columns.
