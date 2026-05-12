package ui

import (
	"errors"

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
		if msg.Err != nil {
			if errors.Is(msg.Err, launchd.ErrProtected) {
				m.flash(msg.Label + ": needs sudo — run `sudo launchctl …` manually")
			} else {
				m.flash(msg.Label + ": " + msg.Action + " failed: " + msg.Err.Error())
			}
		} else {
			m.flash(msg.Label + ": " + msg.Action + " ok")
		}
		// Refresh just that row.
		for i := range m.jobs {
			if m.jobs[i].Label == msg.Label {
				return m, refreshOneCmd(m.jobs[i])
			}
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
	// When the filter box is focused, let it eat almost every key.
	if m.filtering {
		switch msg.String() {
		case "esc":
			m.filtering = false
			m.filterInput.Blur()
			m.filterInput.SetValue("")
			m.applyFilter("")
			return m, nil
		case "enter":
			m.filtering = false
			m.filterInput.Blur()
			m.applyFilter(m.filterInput.Value())
			return m, nil
		}
		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)
		m.applyFilter(m.filterInput.Value())
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
		}
		return m, nil

	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case key.Matches(msg, m.keys.Top):
		m.cursor = 0
		return m, nil

	case key.Matches(msg, m.keys.Bottom):
		if len(m.filtered) > 0 {
			m.cursor = len(m.filtered) - 1
		}
		return m, nil

	case key.Matches(msg, m.keys.PageDn):
		m.cursor += m.height / 2
		if m.cursor >= len(m.filtered) {
			m.cursor = len(m.filtered) - 1
		}
		return m, nil

	case key.Matches(msg, m.keys.PageUp):
		m.cursor -= m.height / 2
		if m.cursor < 0 {
			m.cursor = 0
		}
		return m, nil

	case key.Matches(msg, m.keys.Filter):
		m.filtering = true
		m.filterInput.Focus()
		return m, nil

	case key.Matches(msg, m.keys.Refresh):
		return m, refreshAllCmd(m.jobs)

	case key.Matches(msg, m.keys.Start):
		if j := m.selectedJob(); j != nil {
			return m, runActionCmd("start", *j)
		}

	case key.Matches(msg, m.keys.Stop):
		if j := m.selectedJob(); j != nil {
			return m, runActionCmd("stop", *j)
		}

	case key.Matches(msg, m.keys.Restart):
		if j := m.selectedJob(); j != nil {
			return m, runActionCmd("restart", *j)
		}

	case key.Matches(msg, m.keys.Load):
		if j := m.selectedJob(); j != nil {
			return m, runActionCmd("load", *j)
		}

	case key.Matches(msg, m.keys.Unload):
		if j := m.selectedJob(); j != nil {
			return m, runActionCmd("unload", *j)
		}

	case key.Matches(msg, m.keys.Log):
		return m, m.toggleLog()
	}

	return m, nil
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
