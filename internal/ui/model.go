package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/thonq/launchtui/internal/launchd"
)

// Model is the root bubbletea model. It is mostly a flat record of UI state
// plus the slice of jobs we discovered at startup.
type Model struct {
	keys keyMap

	jobs     []launchd.Job
	status   map[string]launchd.JobStatus
	filtered []int // indices into jobs after applying the filter

	cursor      int
	width       int
	height      int
	showHelp    bool
	showLog     bool
	flashMsg    string
	flashUntil  time.Time
	filterInput textinput.Model
	filtering   bool

	logLines map[string][]string // label → recent lines, oldest first
	tailOut  *logTailer
	tailErr  *logTailer
	tailFor  string // label currently being tailed

	// program is the running bubbletea program — wired in by main() so
	// background goroutines (log tailers) can push messages into it.
	program *tea.Program
}

// SetProgram wires the running tea.Program into the model so background
// goroutines (e.g. log tailers) can push messages.
func (m *Model) SetProgram(p *tea.Program) {
	m.program = p
}

// send is a nil-safe wrapper around program.Send so the tailer goroutines
// can be started before the program is fully wired during early init.
func (m *Model) send(msg tea.Msg) {
	if m.program != nil {
		m.program.Send(msg)
	}
}

// NewModel walks the four launch directories and returns a ready-to-run
// Model. Discovery errors that prevent us from listing anything at all are
// surfaced; a partial read still succeeds (Discover never aborts on a single
// unreadable file).
func NewModel() (*Model, error) {
	jobs, err := launchd.Discover()
	if err != nil {
		return nil, fmt.Errorf("discover: %w", err)
	}

	ti := textinput.New()
	ti.Placeholder = "filter labels…"
	ti.Prompt = "/ "
	ti.CharLimit = 64

	m := &Model{
		keys:        defaultKeyMap(),
		jobs:        jobs,
		status:      make(map[string]launchd.JobStatus, len(jobs)),
		logLines:    make(map[string][]string),
		filterInput: ti,
	}
	m.applyFilter("")
	return m, nil
}

// Init kicks off the first status refresh and starts the 5-second ticker.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		refreshAllCmd(m.jobs),
		tickCmd(),
	)
}

// applyFilter rebuilds m.filtered. An empty query matches everything.
func (m *Model) applyFilter(q string) {
	m.filtered = m.filtered[:0]
	q = strings.ToLower(strings.TrimSpace(q))
	for i, j := range m.jobs {
		if q == "" || fuzzyMatch(strings.ToLower(j.Label), q) {
			m.filtered = append(m.filtered, i)
		}
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

// fuzzyMatch is a trivial in-order substring fuzz: every rune of q must
// appear in s, in order. Good enough for label filtering and keeps the
// dependency list short.
func fuzzyMatch(s, q string) bool {
	if q == "" {
		return true
	}
	i := 0
	for _, r := range q {
		idx := strings.IndexRune(s[i:], r)
		if idx < 0 {
			return false
		}
		i += idx + 1
	}
	return true
}

// selectedJob returns the job under the cursor (or nil if the list is empty).
func (m *Model) selectedJob() *launchd.Job {
	if len(m.filtered) == 0 {
		return nil
	}
	idx := m.filtered[m.cursor]
	return &m.jobs[idx]
}

// flash schedules a transient message at the bottom of the screen.
func (m *Model) flash(msg string) {
	m.flashMsg = msg
	m.flashUntil = time.Now().Add(3 * time.Second)
}

// --- bubbletea command builders ----------------------------------------------

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// statusMsg carries one job's freshly fetched status.
type statusMsg struct {
	Status launchd.JobStatus
	Err    error
}

// actionMsg is dispatched after a start/stop/etc. so we can refresh just
// that one row and show an inline notification.
type actionMsg struct {
	Label  string
	Action string
	Err    error
}

func refreshAllCmd(jobs []launchd.Job) tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(jobs))
	for i := range jobs {
		cmds = append(cmds, refreshOneCmd(jobs[i]))
	}
	return tea.Batch(cmds...)
}

func refreshOneCmd(job launchd.Job) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), launchd.DefaultTimeout)
		defer cancel()
		st, err := launchd.Status(ctx, job)
		return statusMsg{Status: st, Err: err}
	}
}

func runActionCmd(action string, job launchd.Job) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), launchd.DefaultTimeout)
		defer cancel()
		var err error
		switch action {
		case "start":
			err = launchd.Start(ctx, job)
		case "stop":
			err = launchd.Stop(ctx, job)
		case "restart":
			err = launchd.Restart(ctx, job)
		case "load":
			err = launchd.Load(ctx, job)
		case "unload":
			err = launchd.Unload(ctx, job)
		}
		return actionMsg{Label: job.Label, Action: action, Err: err}
	}
}

// appendLog stores a tailer line in the bounded ring for the given job.
func (m *Model) appendLog(label, stream, line string) {
	const maxLines = 500
	entry := stream + " │ " + line
	cur := m.logLines[label]
	cur = append(cur, entry)
	if len(cur) > maxLines {
		cur = cur[len(cur)-maxLines:]
	}
	m.logLines[label] = cur
}
