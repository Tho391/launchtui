# launchtui

> A fast, keyboard-driven terminal UI for `launchd` — a lightweight TUI take on
> [LaunchControl](https://www.soma-zone.com/LaunchControl/).

`launchtui` shows every user agent, global agent and daemon on your Mac in a
single sortable list, lets you start / stop / restart / load / unload them
with one keypress, and tails their `StandardOutPath` / `StandardErrorPath`
log files in a pane. No mouse. No sudo prompts. Single static binary.

---

## Install

### Homebrew (recommended)

```bash
brew install Tho391/tap/launchtui
```

This pulls the latest tagged release from the
[`Tho391/homebrew-tap`](https://github.com/Tho391/homebrew-tap) tap, where the
formula is kept in sync by GoReleaser on every release tag. Pre-built binaries
are published for `darwin/amd64` and `darwin/arm64`.

Upgrade with `brew upgrade launchtui`, uninstall with `brew uninstall launchtui`.

### From source

`launchtui` is written in Go 1.22+ and depends only on
[Charm](https://charm.sh/) (bubbletea / bubbles / lipgloss) and
[howett.net/plist](https://pkg.go.dev/howett.net/plist).

```bash
# If you don't have a Go toolchain installed yet, add one via mise:
mise use -g go@latest

# Install straight from the module path:
go install github.com/thonq/launchtui@latest

# Or build from source:
cd path/to/launchtui
go build -o launchtui .

# Or run straight from source:
go run .
```

If `go` is not on your `$PATH` but you installed it via mise, prefix the
commands with `mise exec -- ` (e.g. `mise exec -- go run .`).

## Keybindings

| Key            | Action                                                |
|----------------|-------------------------------------------------------|
| `j` / `↓`      | move selection down                                   |
| `k` / `↑`      | move selection up                                     |
| `g` / `G`      | jump to first / last visible row                      |
| `Ctrl+D` / `Ctrl+U` | half-page down / up                              |
| `/`            | open fuzzy filter on the label                        |
| `s`            | start selected job (`launchctl kickstart`)            |
| `S`            | stop selected job (`launchctl kill TERM`)             |
| `r`            | restart selected job (`launchctl kickstart -k`)       |
| `L`            | load selected job (`launchctl bootstrap` / `load`)    |
| `U`            | unload selected job (`launchctl bootout` / `unload`)  |
| `l`            | toggle the log-tail pane (Stdout + Stderr)            |
| `R`            | refresh all statuses immediately                      |
| `?`            | toggle the help overlay                               |
| `q` / `Ctrl+C` | quit                                                  |

## Status badges

| Glyph | State      | Meaning                                                       |
|-------|------------|---------------------------------------------------------------|
| `●`   | running    | job has a live PID                                            |
| `○`   | loaded     | bootstrapped into launchd but not currently running           |
| `·`   | stopped    | known to us but not in `launchctl list`                       |
| `✖`   | crashed    | last exit code != 0                                           |
| `⚠`   | throttled  | launchd has flagged "spawn rate limited"                       |
| `◆`   | protected  | lives in a daemon domain — read-only without sudo             |
| `?`   | unknown    | `launchctl` did not report enough to classify                 |

## System daemons (sudo)

`launchtui` deliberately does **not** prompt for `sudo`. Jobs from
`/Library/LaunchDaemons` and `/System/Library/LaunchDaemons` are listed and
badged `◆ protected` but you cannot start/stop them through the UI.

To control them, drop to a shell and run, for example:

```bash
sudo launchctl bootstrap system /Library/LaunchDaemons/com.example.daemon.plist
sudo launchctl bootout    system/com.example.daemon
sudo launchctl kickstart  -k system/com.example.daemon
```

Reload `launchtui` (or press `R`) to see the new status. Note that the
`/System/Library/LaunchDaemons` directory is on the sealed system volume on
Big Sur and newer and cannot be modified at all.

## Screenshot

```
┌─ launchtui ──────────────────────────────────────────────────────────────────┐
│  press / to filter                                       jobs: 12 / 184      │
├─────────────────────────────┬────────────────────────────────────────────────┤
│ ▸ ● com.example.foo         │ Label    com.example.foo                       │
│   ○ com.example.bar         │ Plist    ~/Library/LaunchAgents/…/foo.plist    │
│   ✖ com.example.baz  (78)   │ Domain   user                                  │
│   ⚠ com.example.flap (127)  │ Program  /usr/local/bin/foo --flag             │
│   ◆ com.apple.cfprefsd.xpc  │ State    running                               │
│                             │ PID      12345                                 │
│                             │ LastExit 0                                     │
│                             │ Stdout   ~/Library/Logs/foo.out                │
│                             │ Stderr   ~/Library/Logs/foo.err                │
├─────────────────────────────┴────────────────────────────────────────────────┤
│ j/k move  s start  S stop  r restart  L load  U unload  l log  / filter  ?  │
└──────────────────────────────────────────────────────────────────────────────┘
```

## What's in v0.1

- Enumerate plists from `~/Library/LaunchAgents`, `/Library/LaunchAgents`,
  `/Library/LaunchDaemons` and `/System/Library/LaunchDaemons`.
- Live status via `launchctl print gui/<uid>/<label>` with a
  `launchctl list` fallback.
- Start / Stop / Restart / Load / Unload via the keyboard.
- Tail `StandardOutPath` and `StandardErrorPath` in a side pane.
- Fuzzy filter on label.
- Auto-refresh every 5 seconds.

See [`PLAN.md`](./PLAN.md) for the full feature inventory and v0.2 ideas.

## Development

```bash
go vet ./...      # static checks
go test ./...     # unit tests (fixture-based, see internal/launchd/parse_test.go)
go build ./...    # compile everything
```

All three should pass before opening a commit. See [`AGENTS.md`](./AGENTS.md)
for the full contributor / agent conventions. The same three commands run on
every push and pull request via [`.github/workflows/ci.yml`](.github/workflows/ci.yml).

### Cutting a release

Releases are fully automated via [GoReleaser](https://goreleaser.com/) — see
[`.goreleaser.yml`](./.goreleaser.yml) and
[`.github/workflows/release.yml`](.github/workflows/release.yml).

```bash
git tag v0.1.0
git push --tags
```

The release workflow then:

1. Cross-compiles `darwin/amd64` and `darwin/arm64` binaries with `-trimpath`
   and stamps `main.version`, `main.commit`, and `main.date` via `-ldflags -X`.
2. Bundles each binary with `LICENSE` and `README.md` into a `tar.gz`.
3. Generates `checksums.txt` (SHA-256) and publishes everything as a GitHub
   Release with a Conventional-Commits-filtered changelog.
4. Writes `Formula/launchtui.rb` to
   [`Tho391/homebrew-tap`](https://github.com/Tho391/homebrew-tap), so
   `brew install Tho391/tap/launchtui` picks up the new version on the next
   `brew update`.

Pre-release tags such as `v0.1.0-rc1` are auto-flagged as GitHub prereleases.

#### One-time release setup

Before the first tag push, you need:

1. A public `Tho391/homebrew-tap` repository (empty is fine; GoReleaser
   creates the `Formula/` directory on first run).
2. A repository secret named `HOMEBREW_TAP_TOKEN` containing a Personal
   Access Token with `repo` scope on `Tho391/homebrew-tap` (or a
   fine-grained PAT with `Contents: read & write` scoped only to that repo).

#### Local dry-run

```bash
goreleaser release --snapshot --clean --skip=publish
```

Outputs to `dist/`, doesn't push anywhere — handy for sanity-checking
`.goreleaser.yml` changes before tagging.

## v0.2 and beyond

Intentionally **not** in v0.1 (see PLAN.md for full details):

- Plist editor (per-key validation, expert / XML views).
- WatchPaths / StartCalendarInterval editors.
- Job creation from templates.
- Override-database aware `enable` / `disable`.
- Sudo flow for system-daemon management.
- Crontab import.
- AI assistant.

## License

MIT — see [`LICENSE`](./LICENSE).
