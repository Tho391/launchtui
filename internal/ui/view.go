package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/thonq/launchtui/internal/launchd"
)

// View renders the entire screen. It is composed of:
//   - a header line (title, filter input, counts)
//   - a body split horizontally into list + detail
//   - an optional log pane stacked below the detail
//   - a footer keybinding bar
//   - a centred help overlay when m.showHelp
func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "loading…"
	}
	if m.showHelp {
		return m.renderHelp()
	}

	header := m.renderHeader()
	footer := m.renderFooter()

	bodyHeight := m.height - lipgloss.Height(header) - lipgloss.Height(footer)
	if bodyHeight < 4 {
		bodyHeight = 4
	}

	listWidth := m.width / 2
	if listWidth < 32 {
		listWidth = 32
	}
	if listWidth > 60 {
		listWidth = 60
	}
	detailWidth := m.width - listWidth - 2
	if detailWidth < 20 {
		detailWidth = 20
	}

	list := m.renderList(listWidth, bodyHeight)
	detail := m.renderDetail(detailWidth, bodyHeight)
	body := lipgloss.JoinHorizontal(lipgloss.Top, list, detail)

	screen := lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
	if m.pendingAction != "" {
		modal := m.renderConfirm()
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal, lipgloss.WithWhitespaceChars(" "))
	}
	return screen
}

func (m *Model) renderConfirm() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", styleTitle.Render("confirm: "+m.pendingAction))
	fmt.Fprintf(&b, "%s %s\n\n", styleKey.Render("job"), m.pendingJob.Label)
	fmt.Fprintf(&b, "%s %s\n\n", styleKey.Render("cmd"), m.pendingPreview)
	b.WriteString(styleSubtle.Render("y / Enter confirm   ·   n / Esc cancel"))
	return styleBorder.Padding(1, 2).Render(b.String())
}

func (m *Model) renderHeader() string {
	title := styleTitle.Render("launchtui")
	visible := fmt.Sprintf("%d / %d", len(m.filtered), len(m.jobs))

	badges := []string{"jobs: " + visible}
	if m.statusFilter != filterAll {
		badges = append(badges, "state: "+m.statusFilter.String())
	}
	if m.sortMode != sortByLabel {
		badges = append(badges, "sort: "+m.sortMode.String())
	}
	if m.hideApple && m.hiddenAppleCount > 0 {
		badges = append(badges, fmt.Sprintf("%d hidden (A to show)", m.hiddenAppleCount))
	}
	counts := styleSubtle.Render(strings.Join(badges, "  ·  "))

	var filterView string
	if m.filtering {
		filterView = m.filterInput.View()
	} else if q := m.filterInput.Value(); q != "" {
		filterView = styleSubtle.Render("filter: " + q + "  (esc to clear)")
	} else {
		filterView = styleSubtle.Render("press / to filter")
	}

	spacer := strings.Repeat(" ", maxInt(1, m.width-lipgloss.Width(title)-lipgloss.Width(filterView)-lipgloss.Width(counts)-4))
	return title + "  " + filterView + spacer + counts
}

func (m *Model) renderList(width, height int) string {
	if len(m.filtered) == 0 {
		empty := styleSubtle.Render("no jobs match this filter")
		return styleBorder.Width(width).Height(height).Render(empty)
	}

	// Scroll window around the cursor.
	rows := height - 2
	if rows < 1 {
		rows = 1
	}
	start := m.cursor - rows/2
	if start < 0 {
		start = 0
	}
	end := start + rows
	if end > len(m.filtered) {
		end = len(m.filtered)
		start = maxInt(0, end-rows)
	}

	var lines []string
	for i := start; i < end; i++ {
		job := m.jobs[m.filtered[i]]
		st, ok := m.status[job.Label]
		if !ok {
			st = launchd.JobStatus{Label: job.Label, State: launchd.StateUnknown}
			if job.Domain.Protected() {
				st.State = launchd.StateProtected
			}
		}
		badge := renderBadge(st.State)
		var trailer string
		if st.State == launchd.StateCrashed && st.LastExitCode != 0 {
			trailer = fmt.Sprintf(" (%d)", st.LastExitCode)
		} else if sched := launchd.FormatSchedule(job.Schedule, job.RunAtLoad); sched != "" && st.State == launchd.StateScheduled {
			trailer = " · " + sched
		}
		// Reserve room for the badge, gutter, and trailer so the label
		// truncates first instead of spilling into the detail pane.
		const gutter = 4
		maxLabelWidth := width - gutter - lipgloss.Width(trailer)
		if maxLabelWidth < 8 {
			maxLabelWidth = 8
		}
		label := job.Label
		if len(label) > maxLabelWidth {
			label = label[:maxLabelWidth-1] + "…"
		}
		line := badge + " " + label + styleSubtle.Render(trailer)
		if i == m.cursor {
			line = styleSelected.Render("▸ " + line)
		} else {
			line = "  " + line
		}
		lines = append(lines, line)
	}
	body := strings.Join(lines, "\n")
	return styleBorder.Width(width).Height(height).Render(body)
}

func (m *Model) renderDetail(width, height int) string {
	j := m.selectedJob()
	if j == nil {
		return styleBorder.Width(width).Height(height).Render(styleSubtle.Render("nothing selected"))
	}
	st := m.status[j.Label]

	var b strings.Builder
	row := func(k, v string) {
		fmt.Fprintf(&b, "%s %s\n", styleKey.Render(fmt.Sprintf("%-9s", k)), v)
	}
	row("Label", j.Label)
	row("Plist", j.PlistPath)
	row("Domain", j.Domain.String())
	prog := j.Program
	if len(j.ProgramArgs) > 1 {
		prog = strings.Join(j.ProgramArgs, " ")
	}
	row("Program", prog)
	row("State", renderStateText(st.State))
	pidStr := "—"
	if st.PID > 0 {
		pidStr = fmt.Sprintf("%d", st.PID)
	}
	row("PID", pidStr)
	row("LastExit", fmt.Sprintf("%d", st.LastExitCode))
	row("Disabled", fmt.Sprintf("%v", j.Disabled))
	if sched := launchd.FormatSchedule(j.Schedule, j.RunAtLoad); sched != "" {
		row("Schedule", sched)
	}
	if j.StdoutPath != "" {
		row("Stdout", j.StdoutPath)
	}
	if j.StderrPath != "" {
		row("Stderr", j.StderrPath)
	}
	if st.Message != "" {
		row("Note", st.Message)
	}

	if !m.flashUntil.IsZero() && time.Now().Before(m.flashUntil) {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(colorWarn).Render(m.flashMsg))
	}

	if m.showLog && m.tailFor == j.Label {
		b.WriteString("\n")
		b.WriteString(styleSubtle.Render(strings.Repeat("─", maxInt(8, width-4))))
		b.WriteString("\n")
		b.WriteString(styleKey.Render("log tail (l to close)"))
		b.WriteString("\n")
		// Show only the last (height-currentRowCount) lines so we don't blow
		// past the pane height; lipgloss will clip the rest but truncating
		// keeps rendering snappy.
		lines := m.logLines[j.Label]
		max := height - 16
		if max < 1 {
			max = 1
		}
		if len(lines) > max {
			lines = lines[len(lines)-max:]
		}
		if len(lines) == 0 {
			b.WriteString(styleSubtle.Render("…waiting for output…"))
		} else {
			b.WriteString(strings.Join(lines, "\n"))
		}
	}

	return styleBorder.Width(width).Height(height).Render(b.String())
}

func (m *Model) renderFooter() string {
	parts := []string{
		styleKey.Render("j/k") + " move",
		styleKey.Render("s/S/r") + " start/stop/restart",
		styleKey.Render("L/U") + " load/unload",
		styleKey.Render("o") + " reveal",
		styleKey.Render("e") + " edit",
		styleKey.Render("y") + " yank",
		styleKey.Render("l") + " log",
		styleKey.Render("/") + " filter",
		styleKey.Render("F") + " state",
		styleKey.Render("O") + " sort",
		styleKey.Render("A") + " apple",
		styleKey.Render("?") + " help",
		styleKey.Render("q") + " quit",
	}
	return styleSubtle.Render(strings.Join(parts, "  "))
}

func (m *Model) renderHelp() string {
	bindings := []struct{ k, d string }{
		{"j / ↓", "move down"},
		{"k / ↑", "move up"},
		{"g / G", "jump to top / bottom"},
		{"^D / ^U", "half page down / up"},
		{"/", "filter labels and plist paths"},
		{"F", "cycle status filter (all → running → crashed → throttled → scheduled → protected)"},
		{"O", "cycle sort (label → state → domain → last-exit)"},
		{"A", "toggle Apple / system jobs (hidden by default)"},
		{"s", "start the selected job (with confirmation)"},
		{"S", "stop the selected job (with confirmation)"},
		{"r", "restart the selected job (with confirmation)"},
		{"L", "bootstrap (load) the selected job (with confirmation)"},
		{"U", "bootout (unload) the selected job (with confirmation)"},
		{"o", "reveal plist in Finder (open -R)"},
		{"e", "open plist in $EDITOR (read-only viewer)"},
		{"y", "yank selected job's label to the clipboard"},
		{"l", "toggle log-follow pane"},
		{"R", "refresh statuses immediately"},
		{"?", "toggle this help"},
		{"q / ^C", "quit"},
	}
	var b strings.Builder
	b.WriteString(styleTitle.Render("launchtui — help"))
	b.WriteString("\n\n")
	for _, kb := range bindings {
		fmt.Fprintf(&b, "  %s   %s\n", styleKey.Render(fmt.Sprintf("%-10s", kb.k)), styleHelpDesc.Render(kb.d))
	}
	b.WriteString("\n")
	b.WriteString(styleSubtle.Render("System daemons (◆) are listed read-only. To control them, run\n  `sudo launchctl bootout|bootstrap|kickstart system/<label>` manually."))
	b.WriteString("\n\n")
	b.WriteString(styleSubtle.Render("press ? again to dismiss"))
	return styleBorder.Width(m.width - 4).Render(b.String())
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
