# Active tasks

## GitHub Actions: package + Homebrew distribution

Goal: tag a release with `git tag v0.x.y && git push --tags` and have CI build
the darwin/amd64 + darwin/arm64 binaries, publish a GitHub Release with
checksums, and update the Homebrew formula in `Tho391/homebrew-tap`
automatically. Plus a green-bar CI on every push/PR.

Tooling: [GoReleaser](https://goreleaser.com/) v2 — it's the standard for Go
projects shipping single-binary releases, handles archives + checksums + GH
release + brew formula in one config, and is a CI-time tool only (not a Go
module dep, so no AGENTS.md §"Dependencies" rule violation).

### Plan

- [x] Add `var version/commit/date` + `--version` flag to `main.go`. Required
      so GoReleaser's `-ldflags -X` actually lands somewhere, and so the
      generated brew formula's `test do` block can run a non-TUI command
      that exits cleanly.
- [x] `.goreleaser.yml` — darwin only, amd64 + arm64, `-trimpath` + stripped
      ldflags, reproducible `mod_timestamp`, `tar.gz` archives bundling
      `LICENSE` + `README.md`, sha256 checksums, GitHub release with
      conventional-commit changelog filters, `brews:` block writing
      `Formula/launchtui.rb` to `Tho391/homebrew-tap` via
      `HOMEBREW_TAP_TOKEN`.
- [x] `.github/workflows/ci.yml` — runs `go vet ./...`, `go test ./...`,
      `go build ./...` on `macos-latest` for every push to `main` and every
      pull request. This is the AGENTS.md §"Build & verify" gate enforced
      in CI.
- [x] `.github/workflows/release.yml` — on tag `v*` push, run
      `goreleaser/goreleaser-action@v6 release --clean` on `macos-latest`
      with `GITHUB_TOKEN` + `HOMEBREW_TAP_TOKEN`.
- [x] `README.md` — add a `brew install Tho391/tap/launchtui` Install
      section and a "Cutting a release" subsection.
- [x] Run `go vet ./...`, `go test ./...`, `go build ./...` locally before
      committing. (Cannot run `goreleaser check` without installing it; the
      release workflow will be the authoritative validator on the first
      tag push.)

### One-time human prerequisites (cannot be automated from this repo)

These are blockers for the *first release*, not for landing the workflows.
CI will go green immediately; the release workflow only runs on tag push.

1. Create a public, empty repo `Tho391/homebrew-tap` on GitHub. (Brew
   convention: the `homebrew-` prefix is stripped at install time, so the
   formula is reachable as `Tho391/tap/launchtui`.)
2. Generate a Personal Access Token (classic) with `repo` scope, OR a
   fine-grained PAT scoped only to `Tho391/homebrew-tap` with
   `Contents: read & write`.
3. Add it as `HOMEBREW_TAP_TOKEN` under
   `Tho391/launchtui` → Settings → Secrets and variables → Actions →
   New repository secret.

### Pre-existing inconsistency (flagged, not fixed here)

`go.mod` declares `module github.com/thonq/launchtui` but the repo is
`github.com/Tho391/launchtui`. `go install github.com/thonq/launchtui@latest`
in the README will not resolve. GoReleaser + Homebrew is unaffected (the
release URL comes from `git remote`, not the module path), so this is out
of scope here. Worth a follow-up task: rename the module to
`github.com/Tho391/launchtui` (and fix internal imports), or set up a vanity
import redirect at `thonq/launchtui`. Surface before silently fixing.

### Review (2026-05-12)

- All four files added (`main.go` patch, `.goreleaser.yml`,
  `.github/workflows/ci.yml`, `.github/workflows/release.yml`), README
  patched with Homebrew install + release process.
- Verification gate (`go vet ./... && go test ./... && go build ./...`)
  passes locally. Workflows themselves will only be exercised on push to
  GitHub.
- `--version` flag added to `main.go` is a tiny but real product change —
  surfaced explicitly in the up-front plan and confirmed before doing it.
  Lets `launchtui --version` work and gives the brew formula a clean
  smoke-test target.
- No new Go module deps introduced (`flag` is stdlib). `go.sum` unchanged.

### Follow-up: v0.1.0 first-tag fixup (same day)

The first `v0.1.0` push failed CI with `j.IsAppleSystem undefined`. Root
cause was a narrow `git add` recipe I gave (`git add main.go README.md
tasks/todo.md .goreleaser.yml .github/`) which staged a `main.go` carrying
pre-existing in-flight edits (`runList` calling `Job.IsAppleSystem`)
without the matching definition in `internal/launchd/launchd.go`. The
commit pushed was internally inconsistent.

Fix landed in a follow-up commit:

- Committed the entire 729-line in-flight v0.2 delta (per explicit user
  greenlight): `internal/launchd/{control,discover,launchd,status}.go`,
  `internal/launchd/schedule{,_test}.go` (new), `internal/ui/*.go`,
  `internal/ui/clipboard.go` (new). All verified by
  `go vet ./... && go test ./... && go build ./...`.
- Re-added the lost `var version/commit/date` block + `-v|--version|version`
  arg handler in `main.go`. Without it the `-ldflags -X main.version=...`
  stamping in `.goreleaser.yml` was a silent no-op.
- Migrated `.goreleaser.yml` from the deprecated `brews:` block (deprecated
  since GoReleaser v2.10, will be removed in v3.0) to `homebrew_casks:`.
  User-facing install command unchanged. Added a
  `hooks.post.install` xattr-strip step so Gatekeeper doesn't reject the
  unsigned binary with "launchtui is damaged" on first run.
- Force-retagged `v0.1.0` to the new commit so the release workflow
  re-runs against a clean tree.

Two new lessons captured in `tasks/lessons.md`: (1) never give a narrow
`git add` recipe without re-reading `git status` against the actual
working tree first; (2) tokens never go into chat.

## v0.2 backlog

Pulled from `PLAN.md` §4 ("Stretch goals (v0.2+)"). Promote items into the
**Active tasks** section when work starts.

- [ ] Plist editor: per-key validation, key palette, adaptive UI per job type.
- [ ] Log filtering / grep, regex highlight, jump to next error.
- [ ] Dependency graph across sockets / Mach services / shared programs.
- [ ] System daemon management with opt-in sudo (`sudo -n launchctl …` after an
      explicit per-session permission grant).
- [ ] Templates for common job patterns (every N min, on file change, at login).
- [ ] Override-database awareness — `enable` / `disable` that persists across
      reboots, with a visible toggle in the row.
- [ ] Crontab import / `cron2launchd`-style helper.
- [ ] Theme support (lipgloss palette swap), optional mouse support.

## Feature ideas from pylaunchd + LaunchDeck (research 2026-05-12)

Distilled from a comparison pass against
[pylaunchd](https://github.com/glowinthedark/pylaunchd) (PyQt6 GUI) and
[LaunchDeck](https://github.com/sderosiaux/launchdeck) (Rust TUI). The full
table lives in this chat's transcript; only the prioritized list is captured
here. GUI-only ideas (drag-drop, OS notification popups, dock widgets) and
items that violate the no-sudo / read-only-for-system-daemons posture were
dropped before this list.

Conventions: **S/M/L** effort, **v0.1** = fits MVP per `PLAN.md` §3,
**SCOPE↑** = needs explicit approval before scope expansion (do not start
without sign-off).

### Tier 1 — high value, fits v0.1

- [x] **Add `scheduled` state.** Distinguish loaded-no-PID-with-schedule from
      loaded-no-PID-without-schedule. Today we badge both as `loaded`/`stopped`,
      which makes recurring jobs look idle. Effort: **M**.
      Touches: `internal/launchd/launchd.go` (new `StateScheduled`),
      `internal/launchd/discover.go` (parse `StartInterval` /
      `StartCalendarInterval`), `internal/launchd/parse.go`,
      `internal/ui/styles.go` (new badge), `internal/ui/view.go`.
- [x] **Compact schedule rendering.** Render `StartInterval=300` as `5min`,
      `StartCalendarInterval` as `Sun 09:00`, `00:00`, etc. in the detail
      pane and as a new optional list column. Effort: **M**.
      Touches: `internal/launchd/discover.go`, new
      `internal/launchd/schedule.go` + table tests, `internal/ui/view.go`.
- [x] **Action confirmation modal echoing the literal `launchctl` command.**
      Borrowed from LaunchDeck; pylaunchd's "fire immediately" is the
      anti-pattern. Modal state in the model, `y`/`Enter` confirm, `n`/`Esc`
      cancel. Also fix the flash message to echo the command we just ran.
      Effort: **M**. Touches: `internal/ui/model.go`,
      `internal/ui/update.go`, `internal/ui/view.go`,
      `internal/launchd/control.go` (export the argv we'd run for preview).
- [x] **Status filter cycle.** New key (`F`) cycles
      all → running → crashed → throttled → scheduled → protected. Today
      finding crashed jobs requires fuzzy-typing labels you already know.
      Effort: **S**. Touches: `internal/ui/keys.go`,
      `internal/ui/model.go`, `internal/ui/update.go`, `internal/ui/view.go`.
- [x] **Hide Apple/system jobs by default + `A` toggle.** `com.apple.*` and
      `/System/Library/LaunchDaemons` drown out user jobs (often >90% of the
      list). Default-hide them with a clear "n hidden (A to show)" hint in
      the header. Effort: **S**. Touches: `internal/ui/model.go`,
      `internal/ui/update.go`, `internal/ui/view.go`, `internal/ui/keys.go`.
- [x] **Sort modes cycle (`O`).** label → state → domain → last-exit. Effort:
      **S**. Touches: `internal/ui/model.go`, `internal/ui/update.go`,
      `internal/ui/keys.go`.
- [x] **Stable selection by identity across refresh.** After a status tick
      or filter change, keep the cursor pinned to the same `Label`, not the
      same row index. Today an in-flight refresh that re-sorts could move
      the cursor — latent bug. Effort: **S**. Touches:
      `internal/ui/model.go`, `internal/ui/update.go`.
- [x] **Reveal plist via `open -R <path>`.** One-key (`o`) shell-out to
      Finder; the TUI-friendly version of pylaunchd's "Show in Finder".
      Effort: **S**. Touches: `internal/launchd/control.go` (new
      `RevealInFinder`), `internal/ui/keys.go`, `internal/ui/update.go`.
- [x] **Open plist in `$EDITOR` (`e`).** Already reserved in `PLAN.md`
      keymap. Strictly a *viewer*, not the plist editor — fits v0.1 because
      we're not parsing or writing back. Falls back to `open` if `$EDITOR`
      unset. Suspend/restore the alt screen around the editor invocation.
      Effort: **S**. Touches: `internal/ui/update.go`,
      `internal/ui/keys.go`, possibly `main.go` for `tea.ExecProcess`.
- [x] **Copy to clipboard via `pbcopy`.** `y` copies the selected label.
      Follow-up: detail-pane row focus to copy `PlistPath` /
      `StandardOutPath` / `StandardErrorPath` was deferred — requires new
      detail-row selection state and would expand scope beyond what the
      task brief authorised. Effort: **S**.
      Touches: new `internal/ui/clipboard.go`, `internal/ui/update.go`,
      `internal/ui/keys.go`.
- [x] **Filter on plist path as well as label.** pylaunchd filters on both;
      keep current fuzzy matcher as the label primary and add a secondary
      substring check against `PlistPath`. Avoids users needing to remember
      which domain a job lives in. Effort: **S**. Touches:
      `internal/ui/model.go` (extend `applyFilter`).
- [x] **`launchtui list` CLI subcommand.** Print the discovered inventory
      (label, domain, state, plist path) without entering the TUI; useful
      for scripts and for our own integration tests. `--all` includes
      Apple/system. Effort: **S–M**. Touches: `main.go` (arg parsing) and a
      new `internal/launchd` render helper. Justify in `PLAN.md` if we add
      a flag parsing dep, but `os.Args` + `flag` is in stdlib.

#### Follow-ups discovered while shipping Tier 1

- [ ] **Detail-pane row focus + multi-target yank.** Promote `y` from
      "copy label" to "copy the focused detail row" (`PlistPath`,
      `StandardOutPath`, `StandardErrorPath`). Requires a new detail-pane
      cursor concept (`Tab`/`j`/`k` while focus is in the detail pane) —
      out of the no-scope-creep brief for this batch.
- [ ] **Re-sort on every status tick when `sortMode` depends on state.**
      Currently the list is only re-sorted when the user changes sort
      mode, filter, or toggle. When sorted by `state` / `last-exit`, a job
      whose state changes between ticks keeps its old row position until
      the next user-triggered rebuild. Stable selection by label means the
      cursor will follow correctly once we do re-sort; the missing piece
      is just calling `rebuildList()` after a batch of statusMsgs has
      arrived.

### Tier 2 — high value, fits v0.1 but slightly bigger

- [ ] **Plist health-warning panel.** Pure read-only QuickFix-lite. Flag
      common foot-guns: missing `Program` and empty `ProgramArguments`,
      `KeepAlive=true` with no `Program`, `StandardOutPath` under a
      non-existent dir, `ProgramArguments[0]` not absolute (lesson #1 in
      `tasks/lessons.md`), exit 127 + no `EnvironmentVariables.PATH`.
      Renders in detail; feeds a `W` warnings-only filter.
      Effort: **M**. Touches: new `internal/launchd/health.go` + tests,
      `internal/ui/view.go`, `internal/ui/keys.go`, `internal/ui/model.go`.
- [ ] **Raw `launchctl print` inspect subview.** A togglable view inside
      the detail pane that shows the verbatim `launchctl print
      <target>/<label>` output — what pylaunchd's bottom dock does. Useful
      when our structured fields don't surface what the user needs.
      Effort: **M**. Touches: `internal/launchd/status.go` (expose raw
      print bytes), `internal/ui/model.go`, `internal/ui/view.go`,
      `internal/ui/keys.go`.
- [ ] **Persist UI prefs.** Save last filter, sort, Apple-hidden toggle,
      domain filter to `~/.config/launchtui/state.toml` (or JSON — no new
      dep). pylaunchd does this via QSettings and it's a small thing that
      compounds. Effort: **M**. Touches: new `internal/ui/state.go`,
      `internal/ui/model.go`, `main.go`.

### Tier 3 — scope expansion (NEEDS APPROVAL before starting)

- [ ] **SCOPE↑ Full-screen logs view.** LaunchDeck-style scrollable
      stdout/stderr with Tab to switch streams, `g`/`G` to top/bottom,
      500-line scrollback. Currently `PLAN.md` §4 lists "log filtering /
      grep, regex highlight" as v0.2 — this is the foundation for that.
      Effort: **M–L**. Would touch `internal/ui/logtail.go`,
      `internal/ui/model.go`, `internal/ui/update.go`,
      `internal/ui/view.go`.
- [ ] **SCOPE↑ Homebrew services integration.** Merge `brew services list
      --json` into the inventory and route start/stop/restart through `brew
      services` for those rows. Genuine UX win (LaunchDeck's headline
      feature) but adds a third source of truth and a hard dep on a brew
      binary at runtime — needs explicit greenlight before we widen the
      `internal/launchd` package's contract. Effort: **L**. Would touch
      every file in `internal/launchd/` and require a new `Source` field
      on `Job`.
- [ ] **SCOPE↑ Override-database `enable`/`disable`.** `launchctl
      enable|disable user/<uid>/<label>`. PLAN.md §4 already lists this as
      v0.2 stretch. LaunchDeck has it, pylaunchd fakes it via `load -w`.
      Effort: **M**. Would touch `internal/launchd/control.go` plus row
      rendering for the new "disabled in override DB" badge.
- [ ] **SCOPE↑ `RunAtLoad` toggle from the TUI.** LaunchDeck's `U` toggle
      writes the plist. Touches plist contents → editor territory, which
      `PLAN.md` §3 explicitly excludes. Effort: **S–M** in isolation but
      crosses the v0.1 line.
- [ ] **SCOPE↑ Create LaunchAgent from template (LaunchDeck's `N`).**
      Form-driven plist generation under `~/Library/LaunchAgents` for
      common patterns (every N min, on file change, at login). PLAN.md §3
      explicitly excludes templates. Effort: **L**.
- [ ] **SCOPE↑ Edit / delete plist.** LaunchDeck's `E`/`D`. The `e` viewer
      flow above is fine for v0.1, but a structured editor and a guarded
      delete are v0.2 plist-editor work.
- [ ] **SCOPE↑ Sudo flow for system daemons.** Even LaunchDeck doesn't
      have this yet. Explicitly v0.2 in PLAN.md §4.

### Review (Tier 1 — 2026-05-12)

**What shipped (all 12 Tier 1 items):**

- Group A — `internal/launchd/launchd.go` gained `StateScheduled` +
  `IsAppleSystem()`; `internal/ui/model.go` was refactored around a single
  `rebuildList()` that honours fuzzy label match, plist-path substring,
  hide-Apple, status-filter cycle, sort cycle, and a label-pinned cursor
  via `snapCursor()`/`captureSelection()`. New keys `A`, `F`, `O`.
- Group B — new `internal/launchd/schedule.go` (`FormatSchedule`,
  `ClassifyScheduled`, `formatInterval`, `formatCalendarEvent`) plus
  `schedule_test.go` with 14 `FormatSchedule` cases and 8
  `ClassifyScheduled` cases. `Status()` now upgrades idle-with-schedule
  jobs to `StateScheduled` after both the print and list paths. Detail
  pane shows a `Schedule` row; the list shows the schedule inline next
  to the label for scheduled jobs.
- Group C — `PreviewCommand` / `PreviewCommandString` /
  `RevealInFinder` exported from `internal/launchd/control.go`. Modal
  state lives in the `Model` (`pendingAction`, `pendingJob`,
  `pendingPreview`); when open, the key handler short-circuits and only
  `Confirm` / `Cancel` are heard, satisfying the brief's "modal owns
  `y`" rule. `actionMsg` now carries the preview string so success and
  failure flashes both echo the literal `launchctl` argv. `e` uses
  `tea.ExecProcess` so the alt screen suspends/restores cleanly; falls
  back to `open` when `$EDITOR` is unset. `y` shells to `/usr/bin/pbcopy`
  via the new `internal/ui/clipboard.go`.
- Group D — `main.go` grew a `flag`-based dispatcher. Default behaviour
  unchanged; `launchtui list [-a|--all]` prints
  `label<TAB>domain<TAB>state<TAB>plist_path`. Statuses are fetched in
  parallel with a worker pool of 8 and a 5-second per-call context
  timeout, so a wedged `launchctl` cannot stall the CLI.

**Verification gate:** `go vet ./...`, `go test ./...`, `go build ./...`
all green after each group and at the end. `launchtui list` smoke-tested
against a real machine — 455 jobs total, 33 after `IsAppleSystem` filter,
and real-world jobs like `com.docker.socket` and
`com.google.GoogleUpdater.wake` correctly classified as `scheduled`,
confirming the new state classifier works end-to-end.

**Keymap entries added (no collisions with the existing map):**

- `A` toggle Apple/system jobs · `F` cycle state filter · `O` cycle sort
- `o` reveal plist in Finder · `e` open plist in `$EDITOR` · `y` yank
  label to clipboard
- `n` cancel modal (modal-only; the normal `n` was previously unbound).
  Modal `Confirm` reuses `y` and `Enter`; the handler gates so `y`
  never both opens a modal and copies the label.

**What surprised me:**

- `howett.net/plist` decodes integer keys into a mix of `int64` and
  `uint64`, never plain `int`. A `plistInt` helper in `discover.go` papers
  over this for all of the schedule fields.
- `StartCalendarInterval` can be either a single dict or an array of
  dicts. Both shapes show up in real plists on this machine, so the
  parser handles both.
- Real-world jobs almost never set the `Hour` field without `Minute`
  (and vice versa); the formatter defaults the missing half to `00` so
  `{Weekday: 0, Hour: 9}` cleanly renders as `Sun 09:00` per the spec.
- `weekday 7` is also Sunday in `launchd.plist(5)`. A real plist on the
  test machine used `Weekday: 0`; both are now handled by `weekdayShort`.

**Follow-ups discovered (logged above, not started):**

- Detail-pane row focus for multi-target yank (deferred per the brief).
- Re-sort on status tick for state-dependent sort modes.

**Lessons captured:** appended three entries to `tasks/lessons.md`
(plist integer decoding, `StartCalendarInterval` polymorphism,
`tea.ExecProcess` for shell-outs that need the alt screen).

