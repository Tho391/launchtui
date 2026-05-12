# Lessons learned

Append a bullet here every time the user corrects course. Keep entries terse
(one or two lines). When this file exceeds 100 lines, rotate it to
`lessons_NN.md` and start fresh.

- **launchd exit code 127 = "command not found".** Plists do not inherit the
  user's shell `PATH`. Use absolute paths in `ProgramArguments[0]`, or set
  `EnvironmentVariables.PATH` explicitly in the plist.
