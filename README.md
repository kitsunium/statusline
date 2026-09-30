# statusline

A two-line Powerline status line for the Claude Code CLI, built design-first
with [kit](https://github.com/kitsunium/platform): the structure lives in
`design/`, the Go shells are generated from it, the bodies are written by
hand.

```
  󰙴 󰒍 7 ·2   Opus 5 ●●●●● 34%  2h10  Weekly 61%  3d4h  Opus 48%   42%   …/atlas   feat/x !3 ?2  +12  -3
 Oracle de parité 1/4 ■■■□ Mesurer la latence
```

Line one carries the session: the OS and service health with the MCP
servers, the model with its effort and its quotas, the context window, the
directory the session works in, git and the changes. It condenses to the
width the host gives it. Line two carries the open epics and the update
notice.

## One binary, two processes

The host runs `statusline` on every redraw. It asks a per-instance daemon,
`statusline daemon`, for what is slow or shared — git, the MCP
configuration, transcripts, tasks, the usage API, the status page — and
renders. When no daemon answers, it renders the last snapshot from the cache
and starts one. The daemon polls the usage API at most once a minute,
honours `Retry-After`, stops a minute after the last redraw, and updates the
binary silently from signed releases.

```bash
statusline daemon status   # version, sessions, network bookkeeping
statusline daemon stop
```

## Environment

| Variable | Effect |
|----------|--------|
| `COLUMNS` | width of line one (set by the host; 120 when absent) |
| `STATUSLINE_GLYPHS` | `nerd` (default) or `text` |
| `STATUSLINE_COLORS` | `truecolor` or `256`; default follows `COLORTERM` |
| `STATUSLINE_MCP_LINE` | `2` moves the MCP servers to a pill on line two |
| `STATUSLINE_HIDE` | comma-separated: `context`, `session`, `weekly`, `model`, `credits`, `health` |
| `STATUSLINE_WEIGHTS` | condensing weights, e.g. `path=5,branch=6` |
| `STATUSLINE_LINE_GAP` | blank lines between the two rows, 0-3 |
| `STATUSLINE_ICON_OS` / `_MODEL` / `_PATH` / `_GIT` | `false` hides an icon |
| `STATUSLINE_NO_SELF_UPDATE` | switches the self-update off |

## Development

```bash
make test      # race tests, examples and properties
make parity    # the binary against the legacy product's goldens
make latency   # warm and cold client latency
```

## License

MIT — see [LICENSE](LICENSE).
