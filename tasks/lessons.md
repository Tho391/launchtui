# Lessons learned

Append a bullet here every time the user corrects course. Keep entries terse
(one or two lines). When this file exceeds 100 lines, rotate it to
`lessons_NN.md` and start fresh.

- **launchd exit code 127 = "command not found".** Plists do not inherit the
  user's shell `PATH`. Use absolute paths in `ProgramArguments[0]`, or set
  `EnvironmentVariables.PATH` explicitly in the plist.
- **`howett.net/plist` decodes integers as int64/uint64, never int.** Any
  numeric-key access through `raw["..."]` needs a type-switch helper (see
  `plistInt` in `internal/launchd/discover.go`); a naked `.(int)` assertion
  will silently miss real values.
- **`StartCalendarInterval` is polymorphic.** A single calendar entry is a
  dict; multiple entries are an array of dicts. Both shapes appear in real
  plists on macOS — parsers must handle both. Weekday `0` and `7` both mean
  Sunday per `launchd.plist(5)`.
- **Use `tea.ExecProcess` (not `os/exec` directly) for shell-outs that need
  the terminal.** It suspends/restores Bubble Tea's alt screen around the
  inferior process. Bare `cmd.Run()` from inside an MVU step will render
  garbage over the TUI.
- **Never give a narrow `git add` recipe without running `git status` against
  the *actual* working tree first.** v0.1.0 release shipped a broken commit
  because the recipe staged `main.go` (which had pre-existing in-flight edits
  calling `Job.IsAppleSystem`) without staging `internal/launchd/launchd.go`
  (where the method was defined). When the working tree has uncommitted work
  outside the current task's scope, either: (a) stash it before staging, (b)
  use `git add -p` to inspect hunk-by-hunk, or (c) explicitly enumerate the
  full set after re-reading every modified file. `git add <handful of files>`
  from memory is the bug.
- **Don't paste secrets into chat.** A repo-scoped GitHub PAT was pasted into
  this session and had to be rotated immediately. LLM chats are logged
  upstream; any credential that touches one is compromised. Tokens go into
  GitHub Actions secrets (or the equivalent secret store) via the host UI,
  not into chat, not into source files, not into commit messages.
