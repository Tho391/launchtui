# launchtui — Plan

## 1. Goal

`launchtui` is a fast, keyboard-driven terminal UI for inspecting and controlling
`launchd` jobs on macOS. It is a (much) leaner spiritual sibling of
[soma‑zone's LaunchControl](https://www.soma-zone.com/LaunchControl/) — instead
of a full Cocoa app with editors and palettes, it lives in the terminal and
focuses on the most common day‑to‑day operations: see every agent/daemon on the
machine, know whether it is running, healthy, crashed, or throttled, tail its
logs, and start/stop/restart it without remembering the right `launchctl`
incantation. The user is the same kind of power user LaunchControl targets, but
working over SSH, inside `tmux`, or simply preferring `hjkl` to a mouse.

## 2. Reference: LaunchControl features

Compiled from:

- Product page: <https://www.soma-zone.com/LaunchControl/>
- Manual container: <https://www.soma-zone.com/LaunchControl/manual-container.html>
- FAQ: <https://soma-zone.com/LaunchControl/FAQ.html>
- Release notes: <https://www.soma-zone.com/LaunchControl/ReleaseNotes.html>

### Service management
- Browse every service grouped by domain: User Agents, Global Agents, System
  Agents, User Daemons (rare), System Daemons.
- Show status at a glance: enabled/disabled checkbox, running/stopped, last
  exit code with human-readable explanation on hover.
- Highlight invalid / broken jobs and surface the problem description (QuickFix
  panel).
- Enable / disable (writes to launchd's override database).
- Load / unload (transient — does not touch the override database).
- Ad‑hoc Start / Stop / Restart from the toolbar or menu.
- Bulk operations on groups (QuickLaunch's "control a group with a single click").
- Reveal a job's `.plist` file on disk (Locate / Reveal in Finder).
- Locate in System Settings (for system services backed by settings panes).
- Add a new job (`File ▸ New` ⌘N) with optional Basic Settings template.
- Rename a job (in place, the `.plist` extension is re‑appended automatically).
- Move job definition to Trash (recoverable), with an option to hard delete.
- Import existing `.plist` or `crontab` records — validates and offers fixes
  for broken records in the import dialog.

### Editing
- **Standard Editor** — one control per documented `launchd(8)` key. The UI is
  adaptive: only keys relevant to the selected job/job-type are shown.
- **Expert Editor** — exposes undocumented or experimental keys
  (e.g. `POSIXSpawnType`, feature‑flagged values).
- **XML View** — live preview of the resulting plist as the user edits.
- **Key Palette** (⇧⌘P) — searchable, categorised list of every supported key
  with inline descriptions; drag a key onto the editor to add it.
- Per‑key validation, type checking and error/warning badges with tooltips.
- "QuickFix" actions for known bad states (e.g. empty `StandardOutPath`,
  outdated `fdautil` path after Tahoe 26.1).
- Built‑in awareness of session types (Aqua/Background/LoginWindow/System),
  ACLs and feature flags.

### Logging / debugging
- Dedicated **launchd log panel** filtered to the selected job — what
  `launchd(8)` itself says about loading and running the job.
- **Standard Out** and **Standard Error** log panels that tail the files
  configured in `StandardOutPath` / `StandardErrorPath`.
- Live updates as the log file grows; gracefully handles very large logs and
  non‑UTF8 bytes; detects file deletion / rotation even through symlinks.
- Recommended debug workflow: leave Start unconditional, watch all three panels,
  iterate. Save‑Reload‑Start single‑click on config changes.

### Discovery & search
- Filter the job list by name and/or by property (status, type, etc.).
- Search across all services with "Include System Jobs" toggle.
- Hover tooltips on every exit code with a likely cause.

### Conveniences
- **QuickLaunch** — menu bar extra showing a user‑curated list of jobs with
  their status, supporting load/unload/start/stop and grouped batch actions.
- **fdautil** — companion helper for granting Full Disk Access to scripts
  without breaking macOS' new security model.
- **AI Chat View** — natural‑language analysis and edits of a job's plist
  (OpenAI, Google, Perplexity, OpenRouter, xAI, LM Studio, Ollama).
- Internet Access Policy bundle for Little Snitch.
- Privileged helper tool for operations requiring root (system agents/daemons,
  the override database, fdautil configuration).

### macOS specifics handled by LaunchControl that we should be aware of
- Big Sur+ sealed system volume → system agents/daemons cannot be modified.
- Bookkeeping the launchd override database (enable/disable persists across
  reboots; load/unload does not).
- Difference between `launchctl list` (legacy) and `launchctl print
  gui/<uid>/<label>` / `system/<label>` (modern, structured, more detail).

## 3. MVP scope (v0.1)

Ship the minimum that makes the tool useful for a single sitting of "what's
going on with my agents":

- Enumerate every plist from all four well‑known locations:
  - `~/Library/LaunchAgents` (user agents)
  - `/Library/LaunchAgents` (global agents)
  - `/Library/LaunchDaemons` (global daemons — read‑only without sudo)
  - `/System/Library/LaunchDaemons` (system daemons — read‑only, sealed volume)
- Parse each plist with `howett.net/plist` to get `Label`,
  `ProgramArguments`/`Program`, `StandardOutPath`, `StandardErrorPath`,
  `RunAtLoad`, `KeepAlive`, `Disabled`.
- Query live status via `launchctl print gui/<uid>/<label>` (or
  `system/<label>` for daemons), falling back to `launchctl list | grep <label>`
  if `print` fails.
- Show status with a coloured badge:
  - green ● running (has PID)
  - grey ○ loaded but not running
  - red ✖ crashed (non‑zero last exit code)
  - yellow ⚠ throttled / spawn‑rate limited (exit 127 + throttle hint)
  - blue ◆ protected (system daemon, requires sudo)
- Job actions from the keyboard: `s` start, `S` stop, `r` restart,
  `L` load, `U` unload. After any action, refresh just that row.
- Detail pane on the right: resolved plist path, label, program, last exit
  code, PID, log paths, recent log tail.
- `l` toggles the log‑follow pane — opens `StandardOutPath` and
  `StandardErrorPath` if set and tails new bytes.
- `/` opens a fuzzy filter on the label.
- `?` toggles a help overlay listing every key binding.
- `q` / `Ctrl+C` quits cleanly (cancels tickers, closes file watchers).
- Auto‑refresh status of all visible rows every 5 s via `tea.Tick`.

### Out of scope for v0.1
- Plist editor (standard / expert / XML).
- Watch‑paths, calendar, KeepAlive condition editors.
- Creating new jobs from templates.
- AI chat.
- QuickLaunch menu bar extra.
- Override database (enable/disable persisted across reboots).
- fdautil integration.
- Crontab import.

## 4. Stretch goals (v0.2+)

- Plist editor: per‑key validation, key palette, adaptive UI per job type.
- Log filtering / grep, regex highlight, jump to next error.
- Dependency graph: which jobs reference each other via sockets / Mach
  services, which programs share paths.
- System daemon management with an opt‑in sudo prompt (we shell out to
  `sudo -n launchctl …` after asking permission once per session).
- Templates: `File ▸ New` flow for common patterns (run every N min, run on
  file change, run at login).
- Override database awareness — `enable`/`disable` actions that survive a
  reboot, plus a visible toggle in the row.
- Crontab import / `cron2launchd` style helper.
- Theme support (lipgloss palette swap), mouse support.

## 5. Architecture

```
launchtui/
├── go.mod
├── main.go               // wires bubbletea program
├── PLAN.md
├── README.md
├── LICENSE
├── .gitignore
└── internal/
    ├── launchd/          // pure-Go wrapper around `launchctl` + plist parser
    │   ├── launchd.go    // public types: Job, JobStatus, State
    │   ├── discover.go   // walks the four well-known dirs, parses plists
    │   ├── control.go    // Start/Stop/Restart/Load/Unload
    │   ├── status.go     // Status(label) using `launchctl print`/`list`
    │   ├── parse.go      // parsePrintOutput, parseListLine, parsePlist
    │   └── parse_test.go
    └── ui/               // bubbletea model/view/update
        ├── model.go
        ├── update.go
        ├── view.go
        ├── keys.go
        ├── styles.go
        └── logtail.go    // file tailer producing tea.Msg lines
```

### Key types

```go
// internal/launchd/launchd.go
type Domain int       // UserAgent, GlobalAgent, GlobalDaemon, SystemDaemon
type State  int       // Running, Loaded, Stopped, Crashed, Throttled, Unknown, Protected

type Job struct {
    Label        string
    PlistPath    string
    Domain       Domain
    Program      string
    ProgramArgs  []string
    StdoutPath   string
    StderrPath   string
    Disabled     bool
}

type JobStatus struct {
    Label        string
    State        State
    PID          int
    LastExitCode int
    Message      string  // human-friendly reason
}
```

### Data flow

```
                            +-----------------------+
                            |   discover.Walk()      |
                            |   - parses ~/Library/  |
                            |     and /Library/...   |
                            +-----------+-----------+
                                        |
                                        v
                            +-----------------------+
                            |  []Job  (in-memory)   |
                            +-----------+-----------+
                                        |
              tea.Tick(5s)  +-----------+-----------+   on user action
                  +-------->|   ui.Model.Update     |<-------+
                  |         +-----------+-----------+        |
                  |                     |                    |
                  |                     v                    |
                  |     status.Lookup(label) per row  ---> launchctl print/list
                  |                                          |
                  +----------- updated JobStatus  -----------+
```

### `launchctl print` parsing strategy

`launchctl print gui/<uid>/<label>` returns a free‑form text format that
looks like nested key/value blocks:

```
com.example.foo = {
    active count = 1
    path = /Users/u/Library/LaunchAgents/com.example.foo.plist
    state = running
    pid = 12345
    last exit code = 0
    program = /usr/local/bin/foo
}
```

We do **not** attempt a full structured parse. We grep for the keys we care
about (`state`, `pid`, `last exit code`, `path`, `program`) with line‑oriented
regexes. Whatever Apple changes in future releases we either pick up
automatically (different keys) or fall through to the legacy parser.

Legacy fallback — `launchctl list`:

```
PID	Status	Label
1234	0	com.example.foo
-	0	com.example.bar
```

We tab‑split each non‑header line; `-` PID means loaded‑but‑not‑running,
negative status means signalled, non‑zero means crashed.

## 6. UI layout

```
┌─ launchtui ──────────────────────────────────────────────────────────────────┐
│  /filter: ____                                       jobs: 184  visible: 12  │
├─────────────────────────────┬────────────────────────────────────────────────┤
│ ● com.apple.dock.agent      │ Label  : com.example.foo                       │
│ ○ com.example.foo           │ Plist  : ~/Library/LaunchAgents/…/foo.plist    │
│ ✖ com.example.bar (78)      │ Domain : UserAgent                             │
│ ⚠ com.example.baz (127)     │ Program: /usr/local/bin/foo --flag             │
│ ◆ com.apple.cfprefsd.xpc    │ State  : running    PID: 12345                 │
│   …                         │ Exit   : 0          Disabled: false            │
│                             │ Stdout : ~/Library/Logs/foo.out                │
│                             │ Stderr : ~/Library/Logs/foo.err                │
│                             │────────────────────────────────────────────────│
│                             │ [log tail — toggle with `l`]                   │
│                             │ 12:42:01 starting up                           │
│                             │ 12:42:01 listening on :8080                    │
│                             │ …                                              │
├─────────────────────────────┴────────────────────────────────────────────────┤
│ j/k move  s start  S stop  r restart  L load  U unload  l log  / filter  ?  │
└──────────────────────────────────────────────────────────────────────────────┘
```

### Keymap

| Key            | Action                                          |
|----------------|-------------------------------------------------|
| `j` / `↓`      | move selection down                             |
| `k` / `↑`      | move selection up                               |
| `g` / `G`      | jump to first / last visible row                |
| `Ctrl+D/U`     | half‑page down / up                             |
| `/`            | open fuzzy filter, `Enter` to apply, `Esc` clear|
| `s`            | start selected job                              |
| `S`            | stop selected job                               |
| `r`            | restart selected job                            |
| `L`            | load selected job                               |
| `U`            | unload selected job                             |
| `l`            | toggle log‑follow pane                          |
| `R`            | refresh statuses now (don't wait for tick)      |
| `e`            | edit plist  *(v0.2 — opens `$EDITOR` for now)*  |
| `?`            | toggle help overlay                             |
| `q` / `Ctrl+C` | quit                                            |

## 7. Implementation plan

1. Scaffold project: `go mod init github.com/thonq/launchtui`, add deps
   (`bubbletea`, `bubbles`, `lipgloss`, `howett.net/plist`).
2. `internal/launchd/launchd.go` — types and `Domain` helpers
   (resolve dir → domain, domain → `gui/<uid>` vs `system`).
3. `internal/launchd/discover.go` — walk the four directories, parse each plist
   into a `Job`. Tolerate unreadable files (system daemons may be readable
   without sudo but treat any error as "skip with a note").
4. `internal/launchd/parse.go` — parse `launchctl print` and `launchctl list`
   output into `JobStatus`. Cover throttle / spawn rate hints.
5. `internal/launchd/status.go` — `Status(job)` orchestrates print → fallback to
   list. Always uses a 5s `context.WithTimeout`.
6. `internal/launchd/control.go` — `Start/Stop/Restart/Load/Unload`
   shelling out to `launchctl bootstrap`/`bootout`/`kickstart`/`kill`.
   For protected domains return a typed error the UI can surface.
7. `internal/launchd/parse_test.go` — table tests on fixture strings.
8. `internal/ui/styles.go` — lipgloss palette and badge renderer.
9. `internal/ui/keys.go` — `key.Binding` map.
10. `internal/ui/logtail.go` — a simple tailer goroutine pushing
    `logLineMsg{path, line}` into the tea program.
11. `internal/ui/model.go` — model struct: rows, statuses, filter, help,
    selected idx, tailers, ticker.
12. `internal/ui/update.go` — handle keys, ticks, status results, log lines,
    action results.
13. `internal/ui/view.go` — compose list + detail + (optional) log pane + help.
14. `main.go` — `tea.NewProgram(...).Run()`, AltScreen, mouse off.
15. `README.md`, `LICENSE`, `.gitignore`.
16. `go mod tidy && go vet ./... && go build ./...`.
17. Commit.

## 8. Risks / caveats

- **`launchctl print` is unstable.** Apple explicitly says the output is not a
  stable API. Our parser is intentionally tolerant — missing keys degrade
  gracefully to `Unknown`, and we still have the legacy `list` fallback.
- **Sudo required for system daemons.** We never attempt to control jobs in
  `/Library/LaunchDaemons` or `/System/Library/LaunchDaemons` from the TUI.
  They are listed and badged "protected"; the README documents the manual
  `sudo launchctl …` workflow.
- **Throttled jobs that look like crashes.** A job that exits with 127 within
  10 seconds will be `spawn rate limited` by launchd. We badge those yellow
  (throttled) rather than red (crashed) when we see the hint in
  `launchctl print`'s output.
- **Status drift.** Every 5 s we re‑query *only* the visible jobs (the list
  may be hundreds of rows); we never call `launchctl` per keystroke.
- **Plist quirks.** Some plists use a top‑level `Program` string, others use
  `ProgramArguments`. Some omit a `Label` (we synthesize from the filename).
  Some are binary plist; `howett.net/plist` handles both.
- **Log files may not exist yet** when the job has never run, or may be
  rotated. The tailer reopens on EOF + `os.Stat` mismatch.
- **`launchctl bootout`/`bootstrap` syntax** differs across macOS versions; we
  try the modern domain form first and fall back to legacy `load`/`unload`
  on error.
