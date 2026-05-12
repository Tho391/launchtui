# AGENTS.md — launchtui

Guidance for AI coding agents (and humans) working in this repo. Keep it short,
keep it followed.

## Workflow Orchestration

### 1. Plan Node Default
- Enter plan mode for ANY non-trivial task (3+ steps or architectural decisions).
- If something goes sideways, STOP and re-plan immediately — don't keep pushing.
- Use plan mode for verification steps, not just building.
- Write detailed specs upfront. `PLAN.md` is the source of truth for scope; if a
  task drifts outside it, surface that before coding.

### 2. Subagent Strategy
- This repo is worked on in Cursor. Use Multitask Mode and background subagents
  liberally to keep the main context window clean.
- Offload research, exploration, and parallel analysis to subagents.
- One task per subagent for focused execution.

### 3. Self-Improvement Loop
- After ANY correction from the user: append the pattern to `tasks/lessons.md`.
- If `tasks/lessons.md` exceeds 100 lines, rotate it to `tasks/lessons_NN.md`
  (`lessons_01.md`, `lessons_02.md`, …) and start fresh.
- Review `tasks/lessons*.md` at session start.

### 4. Verification Before Done
- Never mark a task complete without proving it works. See **Build & verify**.
- Diff behavior between `main` and your changes when relevant.
- Ask yourself: "Would a staff engineer approve this?"
- Run tests, check logs, demonstrate correctness.
- Record any new lessons before closing the task.

### 5. Demand Elegance (Balanced)
- For non-trivial changes: pause and ask "is there a more elegant way?"
- If a fix feels hacky: "Knowing everything I know now, implement the elegant
  solution."
- Skip this for simple, obvious fixes — don't over-engineer.

### 6. Autonomous Bug Fixing
- When given a bug report: just fix it. Don't ask for hand-holding.
- Point at logs, errors, failing tests, then resolve them.

## Task Management

1. **Plan first** — write the plan to `tasks/todo.md` with checkable items.
2. **Verify the plan** — check in before starting implementation.
3. **Track progress** — mark items complete as you go.
4. **Explain changes** — high-level summary at each step.
5. **Document results** — add a short review section to `tasks/todo.md`.
6. **Capture lessons** — update `tasks/lessons*.md` after corrections.

## Core Principles

- **Simplicity first.** Make every change as simple as possible. Touch the
  minimum code.
- **No laziness.** Find root causes. No band-aid fixes. Senior dev standards.
- **Minimal impact.** Don't introduce drive-by changes unrelated to the task.

## Project: launchtui

- **Tech stack.** Go ≥ 1.22, [Charm](https://charm.sh/) `bubbletea` / `bubbles`
  / `lipgloss` for the TUI, `howett.net/plist` for plist parsing. No other
  runtime deps.
- **Package layout.**
  - `internal/launchd/` — pure-Go wrapper around `launchctl` plus the plist
    parser. Public types live in `launchd.go`; discovery, control, status, and
    parsing each get their own file.
  - `internal/ui/` — the Bubble Tea MVU (`model.go`, `update.go`, `view.go`,
    plus `keys.go`, `styles.go`, `logtail.go`).
  - Do **not** add new top-level packages without a one-sentence justification
    in `PLAN.md`.
- **Testing.**
  - Parsing logic in `internal/launchd/` MUST have unit tests. Use table-driven
    tests with fixture strings — follow the pattern in
    `internal/launchd/parse_test.go` (const fixtures + `t.Run` subtests).
  - UI code does not need tests in v0.1.
- **macOS-only.** Anything platform-specific is guarded with `//go:build darwin`.
  Don't add Linux/Windows code paths.
- **`launchctl` invocations** always take a `context.Context` with a 5-second
  timeout (`context.WithTimeout(ctx, 5*time.Second)`). Never shell out
  unbounded.
- **Error wrapping.** Wrap with `fmt.Errorf("...: %w", err)`. Use typed errors
  when the UI must branch on the cause (e.g. protected-domain errors).
- **System daemons are read-only.** Jobs from `/Library/LaunchDaemons` and
  `/System/Library/LaunchDaemons` are listed and badged but never controlled
  from the TUI in v0.1 — no in-process sudo elevation.
- **Dependencies.** Don't add a new module dependency without one sentence in
  `PLAN.md` justifying it. Run `go mod tidy` after any change to `go.mod`.
- **Commit messages.** Conventional Commits: `feat:`, `fix:`, `refactor:`,
  `docs:`, `chore:`, `test:`.

## Build & verify

Before declaring any task done, all three of these MUST pass (this is rule #4's
verification gate):

```bash
go vet ./...
go test ./...
go build ./...
```

If you change Go module deps, also run `go mod tidy` and commit the updated
`go.sum`.

## Out of scope for v0.1

See `PLAN.md` §3 ("Out of scope for v0.1"). Do not silently expand scope —
surface the request and wait for a green light. The big ones:

- Plist editor (standard / expert / XML views).
- WatchPaths / StartCalendarInterval editors.
- Job creation from templates.
- Override-database aware `enable` / `disable`.
- In-process sudo for system daemons.
- Crontab import, AI assistant, QuickLaunch menu bar extra.
