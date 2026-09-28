# statusline

Two-line Powerline status line for the host CLI, built design-first with kit
(kitsunium/platform). Successor of kodflow/status-line, which stays frozen at
207dfe2 (D14): nothing is bridged, the parity oracle in `testdata/parity` is
the only link.

## The rule of the repository

`design/` is law on the structure (ADR 0010 of platform): components,
exported types and functions, use cases, ports, which component imports
which, the wiring of each process role. Bodies are free. Until `kit gen` is
published, the files that stand for generated code are named `api.go`
(exported shells delegating to an unexported twin), `port/port.go` and
`roles/*/wire.go`; they are replaced by `*_gen.go` at the first generation.
Anything a component keeps to itself lives in unexported code or in its
`internal/` zone (the renderer: `render/powerline/internal/`).

## Layout

```
design/            product, domains (quota, snapshot, ipc, render, collect),
                   sequences, evidence (observed fixtures), all design/v1
cmd/statusline     one executable: `statusline` = client, `statusline daemon`
roles/{client,daemon}   wiring of each process role (D22)
quota/ snapshot/ ipc/   libraries shared by both roles; ipc = the contract
render/            client: session (stdin), powerline (renderer), show,
                   ctl, cli, daemonlink, systemclock, port
collect/           daemon: state, gather, refresh, update, daemon, and the
                   adapters gitcli, mcpconfig, transcripts, sessions,
                   taskstore, sysinfo, usageapi, credentials, statuspage,
                   statefile, releases, systemclock, port
testdata/parity    legacy goldens, scenarios, latency, harness (own module)
docs/adr           decisions the design's evidence points to
```

## Client and daemon

- Instance = (UID, CLAUDE_CONFIG_DIR or ~/.claude, executable path), under
  `$XDG_RUNTIME_DIR/statusline-<uid>/<digest>/` (0700): `daemon.sock`, the
  SDK file lock, `daemon.pid`, `heartbeat`, `cache/<key>.json`,
  `state/{network,update}.json`, `daemon.log` (256 KiB, one rotation).
- The client dials with a 250 ms budget; no daemon → renders the key's cache
  and starts `statusline daemon`; an older daemon is asked to stop and
  replaced; a newer one is never stopped; a mute one (no answer, heartbeat
  10 s stale) is killed and replaced.
- The daemon: snapshot served as is for 750 ms, else collected; key evicted
  after 60 s; stops when no key is left for 60 s, or after installing an
  update. Usage API at most once a minute, 429 Retry-After persisted,
  exponential back-off to 1 h; status page every 2 min, unknown past 15 min.
- `STATUSLINE_USAGE_URL` / `STATUSLINE_HEALTH_URL` point the daemon at other
  endpoints (the parity harness serves synthetic payloads through them).
- `STATUSLINE_NO_SELF_UPDATE` (or the legacy `STATUS_LINE_NO_SELF_UPDATE`)
  switches the self-update off; a build without `-X main.vendorKey` installs
  nothing.

## Working here

- Every go build/test/vet through `~/.local/bin/lourd`.
- `make test` (race), `make parity` (88 black-box goldens must pass),
  `make latency`.
- Test data is synthetic only: the repository is public.
- Commits: conventional, author Kodflow, no AI attribution (post-commit gate).
