# Parity oracle

The behaviour statusline must reproduce, captured from the legacy product
`kodflow/status-line` at commit `207dfe25e19d117109fe796e84589e4783e2d15a`
(frozen, D14) before any line of the new product existed.

## What is here

| Path | Content |
|------|---------|
| `scenarios/*.json` | 43 synthetic situations: stdin payload, files on disk, git repositories, a host process stand-in, usage and health payloads, environment variants |
| `golden/<scenario>/<variant>.ansi` | exact bytes the legacy binary printed (92 variants) |
| `golden/<scenario>/<variant>.txt` | the same without ANSI escapes, for review |
| `golden/manifest.json` | sha256, exit code and stderr of every golden |
| `latency/legacy-*.json` | legacy client latency, process start to exit, p50/p95/p99 |
| `harness/` | the Go program (own module, stdlib only) that captures and checks |

Everything is synthetic: the repository is public, so no real path, e-mail,
session id, token or transcript appears in a scenario or a golden. The
harness runs each variant in a fresh sandbox (`HOME`, `XDG_RUNTIME_DIR`,
`TMPDIR` under a temporary root), with a closed environment — nothing of the
caller's environment leaks in — and every network access fails fast through a
dead proxy (`HTTPS_PROXY=http://127.0.0.1:9`).

Instants are written relative to the run (`{{epoch:+7830}}`,
`{{rfc3339:-1}}`): countdowns and burn-rate cursors are rendered at minute or
percent granularity, and every offset sits away from a rounding boundary, so
a golden is stable between the capture and a later check. Scenarios with a
pulsing in-progress task run at the start of an even wall-clock second.

## Scope of a golden

- `black-box` — both binaries run through the process boundary; the new one
  must print the same bytes once its daemon is warm.
- `legacy-only` — the situation includes the legacy on-disk cache layout (a
  cache 20 minutes old, an undecodable cached payload). The new daemon owns
  that state, so the golden documents the legacy behaviour and the scenario's
  `note` names the unit test that pins it on the new side.

## Network payloads

- `legacy` flavour: payloads are seeded as the legacy caches
  (`$XDG_RUNTIME_DIR/status-line-usage-<uid>/usage.json`,
  `status-line-health-<uid>/summary.json`) with the scenario's age.
- `kit` flavour: payloads are served by a local HTTP endpoint the new binary
  is pointed at with `STATUSLINE_USAGE_URL` and `STATUSLINE_HEALTH_URL`; the
  harness renders until two consecutive lines agree (warm daemon), then runs
  `statusline daemon stop`.

## Environment covered

`COLUMNS` (absent, garbage, huge, 40–200), `COLORTERM`, `STATUSLINE_COLORS`,
`STATUSLINE_GLYPHS`, `STATUSLINE_MCP_LINE`, `STATUSLINE_HIDE`,
`STATUSLINE_WEIGHTS` (valid and malformed), `STATUSLINE_LINE_GAP` (in and out
of range), `STATUSLINE_ICON_*`, `CLAUDE_CONFIG_DIR`,
`CLAUDE_CODE_TASK_LIST_ID`.

Platforms: goldens are captured on linux/amd64 only. The host command line
(`/proc/<pid>/cmdline`) and the managed MCP path are Linux behaviours; darwin
and windows have no golden.

## Commands

```bash
cd testdata/parity/harness
go build -o /tmp/harness .

# re-capture from the legacy binary (only when the oracle itself is wrong)
/tmp/harness capture -bin <legacy status-line> -flavour legacy

# check the new binary
/tmp/harness check -bin <statusline> -flavour kit

# latency, process start to exit, after a warm-up run
/tmp/harness latency -bin <binary> -flavour kit -runs 300 -scenario busy/default
```

## Legacy latency baseline

Measured on an Intel Core i5-3210M (no AVX2), 300 sequential runs after one
warm-up, linux/amd64:

| Scenario | p50 | p95 | p99 |
|----------|-----|-----|-----|
| `busy/default` (git, MCP, tasks, caches) | 12.0 ms | 21.7 ms | 33.9 ms |
| `degrade-empty-stdin/default` | 8.8 ms | 16.6 ms | 21.6 ms |
