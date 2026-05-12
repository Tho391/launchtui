# launchtui

> A fast, keyboard-driven terminal UI for `launchd` — a lightweight TUI take on
> [LaunchControl](https://www.soma-zone.com/LaunchControl/).

*macOS-only · single Go binary · no sudo, no daemons, no clicks.*

[![CI](https://img.shields.io/github/actions/workflow/status/Tho391/launchtui/ci.yml?branch=main&label=CI&logo=github)](https://github.com/Tho391/launchtui/actions/workflows/ci.yml)
[![Latest release](https://img.shields.io/github/v/release/Tho391/launchtui?logo=github)](https://github.com/Tho391/launchtui/releases/latest)
[![License](https://img.shields.io/github/license/Tho391/launchtui)](./LICENSE)
[![Go version](https://img.shields.io/github/go-mod/go-version/Tho391/launchtui)](./go.mod)
[![Platform](https://img.shields.io/badge/platform-macOS-blue?logo=apple)](https://www.apple.com/macos/)
[![Homebrew](https://img.shields.io/badge/brew-Tho391%2Ftap%2Flaunchtui-FBB040?logo=homebrew)](https://github.com/Tho391/homebrew-tap)

`launchtui` shows every user agent, global agent and daemon on your Mac in a
single sortable list, lets you start / stop / restart / load / unload them
with one keypress, and tails their `StandardOutPath` / `StandardErrorPath`
log files in a pane. No mouse. No sudo prompts. Single static binary.

## Contents

- [Install](#install)
- [Why launchtui?](#why-launchtui)
- [Keybindings](#keybindings)
- [Status badges](#status-badges)
- [System daemons (sudo)](#system-daemons-sudo)
- [Screenshot](#screenshot)
- [What's in v0.1](#whats-in-v01)
- [Development](#development)
- [v0.2 and beyond](#v02-and-beyond)
- [License](#license)

---

## Install

### Homebrew (recommended)

```bash
brew install Tho391/tap/launchtui
```

This pulls the latest tagged release from the
[`Tho391/homebrew-tap`](https://github.com/Tho391/homebrew-tap) tap, where the
cask is kept in sync by GoReleaser on every release tag. Pre-built binaries
are published for `darwin/amd64` and `darwin/arm64`.

Upgrade with `brew upgrade launchtui`, uninstall with `brew uninstall launchtui`.

### From source

`launchtui` is written in Go 1.22+ and depends only on
[Charm](https://charm.sh/) (bubbletea / bubbles / lipgloss) and
[howett.net/plist](https://pkg.go.dev/howett.net/plist) — pulled in
automatically by `go mod download` on first build.

```bash
# If you don't have a Go toolchain yet, install one via mise:
mise use -g go@latest

# Clone the repo:
git clone https://github.com/Tho391/launchtui.git
cd launchtui
```

#### Run locally from source

`go run` recompiles on every invocation — handy while hacking, no binary
left behind:

```bash
go run .                   # start the full TUI
go run . list              # CLI: print every discovered job
                           # (label, domain, state, plist path)
go run . list -a           # ...including Apple/system jobs
go run . --version         # print build info and exit
go run . --help            # usage
```

#### Build a binary

```bash
go build -o launchtui .
./launchtui                # or move it onto your $PATH
```

If you want the same `-ldflags` stamping the release pipeline uses (so
`launchtui --version` shows a real version instead of `dev`):

```bash
go build -trimpath \
  -ldflags "-s -w -X main.version=$(git describe --tags --always) \
            -X main.commit=$(git rev-parse --short HEAD) \
            -X main.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o launchtui .
```

#### Install globally via the Go toolchain

```bash
go install github.com/thonq/launchtui@latest
```

If `go` is not on your `$PATH` but you installed it via mise, prefix the
commands above with `mise exec -- ` (e.g. `mise exec -- go run .`).

---

## Why launchtui?

A handful of other tools manage `launchd`. Here's where `launchtui` fits:

- **[LaunchControl](https://www.soma-zone.com/LaunchControl/)** is a paid
  Cocoa GUI with the gold-standard plist editor. `launchtui` is free,
  keyboard-only, and lives in your terminal next to the rest of your dev
  workflow — it doesn't try to replace LaunchControl's editor.
- **Raw `launchctl`** is fast once you've memorised `bootstrap` /
  `bootout` / `kickstart` and which domain each job lives in. `launchtui`
  shows every agent, every daemon, and every state in one sortable list,
  with one-keypress controls and a live log tail.
- **[pylaunchd](https://github.com/glowinthedark/pylaunchd)** is a PyQt6
  GUI — it needs a Python toolchain and a desktop session. `launchtui`
  ships as a single static Go binary; it drops into a `tmux` pane fine
  and has no runtime to install.
- **[LaunchDeck](https://github.com/sderosiaux/launchdeck)** is a Rust TUI
  with Homebrew-services integration and a built-in plist editor.
  `launchtui` is more opinionated about staying read-only on system
  daemons and never elevating in-process — `sudo` is something you type
  yourself, in your own shell.
- **No background process, no privileged helper, no `.app` bundle.**
  Install with `brew`, delete with `brew uninstall`. That's the whole
  surface area.

## Keybindings

| Key            | Action                                                |
|----------------|-------------------------------------------------------|
| `j` / `↓`      | move selection down                                   |
| `k` / `↑`      | move selection up                                     |
| `g` / `G`      | jump to first / last visible row                      |
| `Ctrl+D` / `Ctrl+U` | half-page down / up                              |
| `/`            | open fuzzy filter on the label or plist path          |
| `F`            | cycle the state filter (all → running → crashed → …)  |
| `O`            | cycle the sort mode (label → state → domain → exit)   |
| `A`            | toggle Apple/system jobs in the list (hidden by default) |
| `s`            | start selected job (`launchctl kickstart`)            |
| `S`            | stop selected job (`launchctl kill TERM`)             |
| `r`            | restart selected job (`launchctl kickstart -k`)       |
| `L`            | load selected job (`launchctl bootstrap` / `load`)    |
| `U`            | unload selected job (`launchctl bootout` / `unload`)  |
| `l`            | toggle the log-tail pane (Stdout + Stderr)            |
| `o`            | reveal selected plist in Finder (`open -R`)           |
| `e`            | view selected plist in `$EDITOR`                      |
| `y`            | yank the selected label to the clipboard (`pbcopy`)   |
| `T`            | cycle the colour theme (Aurora → Mocha → High-Contrast) |
| `R`            | refresh all statuses immediately                      |
| `?`            | toggle the help overlay                               |
| `q` / `Ctrl+C` | quit                                                  |

When an action would mutate state (start / stop / restart / load / unload),
a confirmation modal echoes the literal `launchctl` command before running.
`y` / `Enter` confirms, `n` / `Esc` cancels.

## Status badges

| Badge | State | Meaning |
| --- | --- | --- |
| ![running](https://img.shields.io/badge/%E2%97%8F_running-3FB950?style=flat-square) | running | job has a live PID |
| ![loaded](https://img.shields.io/badge/%E2%97%8B_loaded-8b949e?style=flat-square) | loaded | bootstrapped into launchd but not currently running |
| ![scheduled](https://img.shields.io/badge/%E2%97%B7_scheduled-79C0FF?style=flat-square) | scheduled | idle but has a `StartInterval` / `StartCalendarInterval` / `RunAtLoad` |
| ![stopped](https://img.shields.io/badge/%C2%B7_stopped-8b949e?style=flat-square) | stopped | known to us but not in `launchctl list` |
| ![crashed](https://img.shields.io/badge/%E2%9C%96_crashed-F85149?style=flat-square) | crashed | last exit code != 0 |
| ![throttled](https://img.shields.io/badge/%E2%9A%A0_throttled-E3B341?style=flat-square) | throttled | launchd has flagged "spawn rate limited" |
| ![protected](https://img.shields.io/badge/%E2%97%86_protected-58A6FF?style=flat-square) | protected | lives in a daemon domain — read-only without sudo |
| ![unknown](https://img.shields.io/badge/%3F_unknown-6e7681?style=flat-square) | unknown | `launchctl` did not report enough to classify |

The colors above match the lipgloss palette in
[`internal/ui/styles.go`](./internal/ui/styles.go), so the chips render
roughly the same hue as the TUI's in-terminal glyphs.

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

```text
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
4. Writes `Casks/launchtui.rb` to
   [`Tho391/homebrew-tap`](https://github.com/Tho391/homebrew-tap) (via the
   GoReleaser `homebrew_casks:` block — the deprecated `brews:` formula
   block was removed during the v0.1.0 fixup), so
   `brew install Tho391/tap/launchtui` picks up the new version on the next
   `brew update`.

Pre-release tags such as `v0.1.0-rc1` are auto-flagged as GitHub prereleases.

#### One-time release setup

Before the first tag push, you need:

1. A public `Tho391/homebrew-tap` repository (empty is fine; GoReleaser
   creates the `Casks/` directory on first run).
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

---

## License

MIT — see [`LICENSE`](./LICENSE).
