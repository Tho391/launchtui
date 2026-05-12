package ui

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/thonq/launchtui/internal/launchd"
)

// Update is the central bubbletea reducer. It is intentionally one big switch
// because the model is small and the indirection of dispatch tables is more
// pain than payoff at this size.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case statusMsg:
		if msg.Status.Label != "" {
			m.status[msg.Status.Label] = msg.Status
		}
		return m, nil

	case actionMsg:
		preview := msg.Preview
		if preview == "" {
			preview = msg.Action
		}
		if msg.Err != nil {
			if errors.Is(msg.Err, launchd.ErrProtected) {
				m.flash(msg.Label + ": needs sudo — run `sudo launchctl …` manually")
			} else {
				m.flash(msg.Label + ": `" + preview + "` failed: " + msg.Err.Error())
			}
		} else {
			m.flash(msg.Label + ": ran `" + preview + "`")
		}
		// Refresh just that row.
		for i := range m.jobs {
			if m.jobs[i].Label == msg.Label {
				return m, refreshOneCmd(m.jobs[i])
			}
		}
		return m, nil

	case revealMsg:
		if msg.Err != nil {
			m.flash("reveal failed: " + msg.Err.Error())
		} else {
			m.flash("revealed " + msg.Path)
		}
		return m, nil

	case clipboardMsg:
		if msg.Err != nil {
			m.flash("yank failed: " + msg.Err.Error())
		} else {
			m.flash("copied: " + msg.Value)
		}
		return m, nil

	case editorDoneMsg:
		if msg.Err != nil {
			m.flash("editor exited: " + msg.Err.Error())
		}
		return m, nil

	case logLineMsg:
		if msg.Label == m.tailFor {
			m.appendLog(msg.Label, msg.Stream, msg.Line)
		}
		return m, nil

	case logStoppedMsg:
		// Nothing to do — tail goroutine has exited cleanly.
		return m, nil

	case tickMsg:
		return m, tea.Batch(refreshAllCmd(m.jobs), tickCmd())
	}

	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Modal owns every key while it is open. y / Enter confirm; n / Esc
	// cancel; everything else is ignored so the user cannot accidentally
	// trigger a second action mid-confirmation.
	if m.pendingAction != "" {
		switch {
		case key.Matches(msg, m.keys.Confirm):
			action := m.pendingAction
			job := m.pendingJob
			preview := m.pendingPreview
			m.pendingAction = ""
			m.pendingPreview = ""
			return m, runActionCmd(action, job, preview)
		case key.Matches(msg, m.keys.Cancel):
			m.pendingAction = ""
			m.pendingPreview = ""
			return m, nil
		}
		return m, nil
	}

	// When the filter box is focused, let it eat almost every key.
	if m.filtering {
		switch msg.String() {
		case "esc":
			m.filtering = false
			m.filterInput.Blur()
			m.filterInput.SetValue("")
			m.rebuildList()
			return m, nil
		case "enter":
			m.filtering = false
			m.filterInput.Blur()
			m.rebuildList()
			return m, nil
		}
		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)
		m.rebuildList()
		return m, cmd
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		m.tailOut.stop()
		m.tailErr.stop()
		return m, tea.Quit

	case key.Matches(msg, m.keys.Help):
		m.showHelp = !m.showHelp
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
			m.captureSelection()
		}
		return m, nil

	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
			m.captureSelection()
		}
		return m, nil

	case key.Matches(msg, m.keys.Top):
		m.cursor = 0
		m.captureSelection()
		return m, nil

	case key.Matches(msg, m.keys.Bottom):
		if len(m.filtered) > 0 {
			m.cursor = len(m.filtered) - 1
			m.captureSelection()
		}
		return m, nil

	case key.Matches(msg, m.keys.PageDn):
		m.cursor += m.height / 2
		if m.cursor >= len(m.filtered) {
			m.cursor = len(m.filtered) - 1
		}
		m.captureSelection()
		return m, nil

	case key.Matches(msg, m.keys.PageUp):
		m.cursor -= m.height / 2
		if m.cursor < 0 {
			m.cursor = 0
		}
		m.captureSelection()
		return m, nil

	case key.Matches(msg, m.keys.Filter):
		m.filtering = true
		m.filterInput.Focus()
		return m, nil

	case key.Matches(msg, m.keys.ToggleApple):
		m.hideApple = !m.hideApple
		m.rebuildList()
		if m.hideApple {
			m.flash("hiding Apple / system jobs (A to show)")
		} else {
			m.flash("showing all jobs")
		}
		return m, nil

	case key.Matches(msg, m.keys.StatusFilter):
		m.statusFilter = (m.statusFilter + 1) % 6
		m.rebuildList()
		m.flash("filter: " + m.statusFilter.String())
		return m, nil

	case key.Matches(msg, m.keys.SortCycle):
		m.sortMode = (m.sortMode + 1) % 4
		m.rebuildList()
		m.flash("sort: " + m.sortMode.String())
		return m, nil

	case key.Matches(msg, m.keys.Theme):
		m.themeIdx = (m.themeIdx + 1) % len(allThemes)
		m.theme = allThemes[m.themeIdx]
		m.flash("theme: " + m.theme.Name)
		return m, nil

	case key.Matches(msg, m.keys.Refresh):
		return m, refreshAllCmd(m.jobs)

	case key.Matches(msg, m.keys.Start):
		m.openConfirm("start")
		return m, nil

	case key.Matches(msg, m.keys.Stop):
		m.openConfirm("stop")
		return m, nil

	case key.Matches(msg, m.keys.Restart):
		m.openConfirm("restart")
		return m, nil

	case key.Matches(msg, m.keys.Load):
		m.openConfirm("load")
		return m, nil

	case key.Matches(msg, m.keys.Unload):
		m.openConfirm("unload")
		return m, nil

	case key.Matches(msg, m.keys.Reveal):
		if j := m.selectedJob(); j != nil {
			return m, revealCmd(j.PlistPath)
		}

	case key.Matches(msg, m.keys.Edit):
		if j := m.selectedJob(); j != nil {
			return m, editPlistCmd(j.PlistPath)
		}

	case key.Matches(msg, m.keys.Yank):
		if j := m.selectedJob(); j != nil {
			return m, yankCmd(j.Label)
		}

	case key.Matches(msg, m.keys.Log):
		return m, m.toggleLog()
	}

	return m, nil
}

// openConfirm queues a modal asking the user to approve the given action
// against the currently selected job. The literal launchctl argv we'd run
// is rendered inside the modal so there are no surprises.
func (m *Model) openConfirm(action string) {
	j := m.selectedJob()
	if j == nil {
		return
	}
	preview := launchd.PreviewCommandString(action, *j)
	if preview == "" {
		m.flash(action + ": no service target for " + j.Label)
		return
	}
	m.pendingAction = action
	m.pendingJob = *j
	m.pendingPreview = preview
}

// --- shell-out commands ------------------------------------------------------

type revealMsg struct {
	Path string
	Err  error
}

func revealCmd(path string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), launchd.DefaultTimeout)
		defer cancel()
		err := launchd.RevealInFinder(ctx, path)
		return revealMsg{Path: path, Err: err}
	}
}

type clipboardMsg struct {
	Value string
	Err   error
}

func yankCmd(value string) tea.Cmd {
	return func() tea.Msg {
		err := copyToClipboard(value)
		return clipboardMsg{Value: value, Err: err}
	}
}

type editorDoneMsg struct {
	Err error
}

// editPlistCmd opens the given plist in $EDITOR, falling back to `open` when
// EDITOR is unset. tea.ExecProcess takes care of suspending and restoring
// the alt screen around the inferior process.
func editPlistCmd(path string) tea.Cmd {
	editor := strings.TrimSpace(os.Getenv("EDITOR"))
	var c *exec.Cmd
	if editor == "" {
		c = exec.Command("/usr/bin/open", path)
	} else {
		parts := strings.Fields(editor)
		c = exec.Command(parts[0], append(parts[1:], path)...)
	}
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return editorDoneMsg{Err: err}
	})
}

// toggleLog flips the log-follow pane. When enabling, it starts one tailer
// per non-empty log path on the selected job; when disabling, it cancels them.
func (m *Model) toggleLog() tea.Cmd {
	if m.showLog {
		m.tailOut.stop()
		m.tailErr.stop()
		m.tailOut = nil
		m.tailErr = nil
		m.tailFor = ""
		m.showLog = false
		return nil
	}
	j := m.selectedJob()
	if j == nil {
		return nil
	}
	m.showLog = true
	m.tailFor = j.Label
	m.logLines[j.Label] = nil
	if j.StdoutPath != "" {
		m.tailOut = startLogTailer(j.Label, "out", j.StdoutPath, m.send)
	}
	if j.StderrPath != "" {
		m.tailErr = startLogTailer(j.Label, "err", j.StderrPath, m.send)
	}
	if m.tailOut == nil && m.tailErr == nil {
		m.appendLog(j.Label, "—", "no StandardOutPath / StandardErrorPath configured")
	}
	return nil
}
