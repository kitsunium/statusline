# statusline

Two-line Powerline status line for the host CLI, built design-first with kit
(kitsunium/platform). Successor of kodflow/status-line, which stays frozen at
207dfe2 (D14): nothing is bridged, the parity oracle in `testdata/parity` is
the only link.

## The rule of the repository

`design/` is law on the structure (ADR 0010 of platform): components,
exported types and functions, use cases, ports, which component imports
which, the links each body makes, the wiring of each process role. Bodies
are free. `kit gen` (platform, version pinned by `kit.version`, read by the
CI and by kit's own toolchain switch) writes
`*_gen.go` / `*_gen_test.go` — never edit them — and, once, the skeleton of
each hand-written twin in `<name>.go`. Twins live in those files: `kit gen`
recreates a missing one even when the twin sits elsewhere. `kit gen -check`
and `lourd kit check` must stay green; `closed: true` with no exceptions.

- Every implemented method or function carries a named property
  (`properties:` in the design, `property<Type><Method><Name>` in
  `properties_test.go` or `more_properties_test.go`, with rapid).
- Every scenario is a `given<UseCase><Scenario>` in
  `<usecase>_scenarios_test.go`; every sequence a `givenSequence<Name>`.
- Named non-struct types are values with `underlying:` (quota `Kind`,
  `Source`, `Level`; snapshot `MCPSource`, `Health`, `OS`, `MCPServers`).
- The build identity (`version`, `vendorKey`) is declared under
  `binaries[].build`: `-ldflags -X main.version=… -X main.vendorKey=…`; the
  adapters that need it take `build:version` in their `uses`.
- `statusline daemon status|stop` belong to the daemon role (the command
  word `daemon` selects it); they ask the instance's daemon through the
  contract.
- Adapters take only what the design's `uses` gives their constructor;
  configuration (instance paths, endpoints) is read in the body.
- Anything a component keeps to itself lives in unexported code or its
  `internal/` zone (the renderer: `render/powerline/internal/`).

## Layout

```
design/            product.yaml (binary, roles, contract, evidence),
                   status.yaml (quota, snapshot, ipc libraries),
                   render.yaml (client), collect.yaml (daemon)
cmd/statusline     main_gen.go + client/ and daemon/ wiring (generated, D22)
quota/ snapshot/ ipc/   libraries shared by both roles; ipc = the contract
render/            session, powerline, show (use cases + ports), cli,
                   daemonlink, systemclock
collect/           state, collector (use cases + ports), daemon (Listener:
                   entry + contract), gitcli, mcpconfig, transcripts,
                   sessions, taskstore, sysinfo, usageapi, credentials,
                   statuspage, statefile, sessioncache, releases, wallclock
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
  switches the self-update off. A release sets `-X main.version=` and
  `-X main.vendorKey=` (`make build VERSION=… VENDOR_KEY=…`); without a key
  nothing is installed; without a version (`dev`) nothing is even checked.

## Working here

- Every go build/test/vet through `~/.local/bin/lourd`.
- `make test` (race), `make design` (kit gen -check, kit check), `kit test`
  (tests per component),
  `make parity` (88 black-box goldens must pass), `make latency`.
- `kit harness install` writes the kit hooks locally (generated files
  refused to Write/Edit, a digest check on Stop); they are never committed,
  nor is any entry for them in .gitignore — the CI is the authority (D7).
- Test data is synthetic only: the repository is public.
- Commits: conventional, author Kodflow, no AI attribution (post-commit gate).
