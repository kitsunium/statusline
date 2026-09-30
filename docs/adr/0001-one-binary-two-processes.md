# ADR 0001 — One binary, two processes

- **Status**: Accepted
- **Date**: 2026-09-28
- **Origin**: plan kit-design-first v7, D5, D11, D14, D22, §9

## Context

The host runs the status line on every redraw, up to once a second per
session. The legacy product (kodflow/status-line) did everything in that
process: git, file reads, and a detached child per stale cache to refresh the
usage API and the status page. With several sessions open, each redraw
re-read every source, and the usage endpoint was polled by as many detached
children as there were stale renders — the endpoint answers 429 when polled
too often, and the legacy product had no memory of it.

## Decision

- One executable, two process roles: `statusline` (the client, what the host
  runs) and `statusline daemon`, started by a client with `os.Executable()`.
- One daemon per (UID, host configuration directory, executable path), under
  a file lock; its socket, lock, cache and state live in a private (0700)
  directory under `$XDG_RUNTIME_DIR` (else the temporary directory).
- The daemon keeps state per session key {session, transcript, working
  directory}, evicts a key unused for 60 s, and stops when none is left.
- The client and the daemon only talk through `statusline.ipc/v1`
  (length-prefixed JSON). Each sends Hello; the client asks an **older**
  daemon to stop and starts its own; it **never stops a newer one** and
  renders from the cache instead.
- The daemon touches a heartbeat file every second and records its pid. A
  daemon that accepts a connection but does not answer within the client's
  budget while its heartbeat is ten seconds stale is mute: the client kills
  it (the kernel releases its lock) and starts a new one. A slow daemon with
  a fresh heartbeat is left alone.
- A client that cannot reach a warm daemon renders the key's cached snapshot
  (written by the daemon after each collection) in at most 50 ms, and starts
  the daemon for the next redraw.
- The usage API is polled at most once per minute per instance; a 429 records
  its Retry-After (else an exponential back-off from one minute to one hour)
  in the persisted state, which a restarted daemon honours. The OAuth token
  is read on every call, never refreshed, never cached, never sent anywhere
  else.
- The socket accepts only a peer with the daemon's own UID (SO_PEERCRED on
  Linux, the 0700 directory elsewhere until the SDK ships the portable
  primitive).
- Silent self-update (D11) is the daemon's: SDK selfupdate, `<bin>.prev`
  kept, the new binary probed, a failed probe rolls back and records
  `bad_version`; never any elevation. After an install the daemon stops; the
  next client starts the new binary.

## Consequences

- A warm redraw is one dial and one frame; git and the configuration files
  are read by one process for all sessions of the instance.
- The first redraw after a start, an update or an eviction shows the cached
  line, which may be up to one refresh old.
- Two binaries at two paths (a release and a development build) are two
  instances and never share a daemon.
