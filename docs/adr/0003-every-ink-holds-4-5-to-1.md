# ADR 0003 — Every ink holds 4.5:1 on its ground

- **Status**: Accepted (carried over from kodflow/status-line, frozen at 207dfe2)
- **Date**: 2026-09-28

## Decision

Pale xterm-256 grounds, a dark ink of the same hue; every ink/ground pair
holds at least 4.5:1, checked by a test that also reads 24-bit escapes. Model
pills use 24-bit inks near 5.4:1 with a 256-colour fallback chosen once per
process (`COLORTERM`, `STATUSLINE_COLORS`). Two adjacent segments never share
a close ground.
