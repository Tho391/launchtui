package ui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/thonq/launchtui/internal/launchd"
)

// sortMode is the user-cycled ordering for the visible job list.
type sortMode int

const (
	sortByLabel sortMode = iota
	sortByState
	sortByDomain
	sortByLastExit
)

func (s sortMode) String() string {
	switch s {
	case sortByState:
		return "state"
	case sortByDomain:
		return "domain"
	case sortByLastExit:
		return "last-exit"
	default:
		return "label"
	}
}

// statusFilter is the user-cycled state restriction applied on top of the
// text filter.
type statusFilter int

const (
	filterAll statusFilter = iota
	filterRunning
	filterCrashed
	filterThrottled
	filterScheduled
	filterProtected
)

func (f statusFilter) String() string {
	switch f {
	case filterRunning:
		return "running"
	case filterCrashed:
		return "crashed"
	case filterThrottled:
		return "throttled"
	case filterScheduled:
		return "scheduled"
	case filterProtected:
		return "protected"
	default:
		return "all"
	}
}

// Model is the root bubbletea model. It is mostly a flat record of UI state
// plus the slice of jobs we discovered at startup.
type Model struct {
	keys keyMap

	jobs     []launchd.Job
	status   map[string]launchd.JobStatus
	filtered []int // indices into jobs after applying the filter

	cursor        int
	selectedLabel string // pinned identity so refresh / sort can re-snap the cursor
	width         int
	height        int
	showHelp      bool
	showLog       bool
	flashMsg      string
	flashUntil    time.Time
	filterInput   textinput.Model
	filtering     bool

	// View knobs cycled by A / F / O — all persist for the session only.
	hideApple        bool
	hiddenAppleCount int
	statusFilter     statusFilter
	sortMode         sortMode

	// Confirmation modal for control actions. When pendingAction is
	// non-empty the modal is visible and the key handler is gated so the
	// modal owns y / n / Enter / Esc.
	pendingAction  string
	pendingJob     launchd.Job
	pendingPreview string

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
		hideApple:   true,
	}
	m.rebuildList()
	return m, nil
}

// Init kicks off the first status refresh and starts the 5-second ticker.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		refreshAllCmd(m.jobs),
		tickCmd(),
	)
}

// rebuildList rebuilds m.filtered from the current text filter, status
// filter cycle, Apple-hidden toggle, and sort mode. The cursor is re-snapped
// to m.selectedLabel so user actions stay pinned to the same job across
// refreshes, filter changes, and sort cycles.
func (m *Model) rebuildList() {
	q := strings.ToLower(strings.TrimSpace(m.filterInput.Value()))
	m.filtered = m.filtered[:0]
	m.hiddenAppleCount = 0
	for i := range m.jobs {
		j := &m.jobs[i]
		if !m.passesTextAndStatus(j, q) {
			continue
		}
		if m.hideApple && j.IsAppleSystem() {
			m.hiddenAppleCount++
			continue
		}
		m.filtered = append(m.filtered, i)
	}
	m.sortFiltered()
	m.snapCursor()
}

func (m *Model) passesTextAndStatus(j *launchd.Job, q string) bool {
	if m.statusFilter != filterAll {
		st := m.statusForJob(j)
		var target launchd.State
		switch m.statusFilter {
		case filterRunning:
			target = launchd.StateRunning
		case filterCrashed:
			target = launchd.StateCrashed
		case filterThrottled:
			target = launchd.StateThrottled
		case filterScheduled:
			target = launchd.StateScheduled
		case filterProtected:
			target = launchd.StateProtected
		}
		if st.State != target {
			return false
		}
	}
	if q == "" {
		return true
	}
	if fuzzyMatch(strings.ToLower(j.Label), q) {
		return true
	}
	if strings.Contains(strings.ToLower(j.PlistPath), q) {
		return true
	}
	return false
}

// statusForJob returns the live status for a job, falling back to a
// domain-appropriate default when we have not heard back from launchctl yet.
func (m *Model) statusForJob(j *launchd.Job) launchd.JobStatus {
	if st, ok := m.status[j.Label]; ok {
		return st
	}
	st := launchd.JobStatus{Label: j.Label, State: launchd.StateUnknown}
	if j.Domain.Protected() {
		st.State = launchd.StateProtected
	}
	return st
}

func (m *Model) sortFiltered() {
	switch m.sortMode {
	case sortByLabel:
		sort.SliceStable(m.filtered, func(i, k int) bool {
			return strings.ToLower(m.jobs[m.filtered[i]].Label) <
				strings.ToLower(m.jobs[m.filtered[k]].Label)
		})
	case sortByState:
		sort.SliceStable(m.filtered, func(i, k int) bool {
			ai := stateRank(m.statusForJob(&m.jobs[m.filtered[i]]).State)
			ak := stateRank(m.statusForJob(&m.jobs[m.filtered[k]]).State)
			if ai != ak {
				return ai < ak
			}
			return strings.ToLower(m.jobs[m.filtered[i]].Label) <
				strings.ToLower(m.jobs[m.filtered[k]].Label)
		})
	case sortByDomain:
		sort.SliceStable(m.filtered, func(i, k int) bool {
			di := m.jobs[m.filtered[i]].Domain
			dk := m.jobs[m.filtered[k]].Domain
			if di != dk {
				return di < dk
			}
			return strings.ToLower(m.jobs[m.filtered[i]].Label) <
				strings.ToLower(m.jobs[m.filtered[k]].Label)
		})
	case sortByLastExit:
		sort.SliceStable(m.filtered, func(i, k int) bool {
			ai := m.statusForJob(&m.jobs[m.filtered[i]]).LastExitCode
			ak := m.statusForJob(&m.jobs[m.filtered[k]]).LastExitCode
			// Non-zero first; among non-zero, larger code first.
			if (ai == 0) != (ak == 0) {
				return ai != 0
			}
			if ai != ak {
				return ai > ak
			}
			return strings.ToLower(m.jobs[m.filtered[i]].Label) <
				strings.ToLower(m.jobs[m.filtered[k]].Label)
		})
	}
}

// stateRank orders states for sortByState — things the user is most likely
// looking at first.
func stateRank(s launchd.State) int {
	switch s {
	case launchd.StateCrashed:
		return 0
	case launchd.StateThrottled:
		return 1
	case launchd.StateRunning:
		return 2
	case launchd.StateScheduled:
		return 3
	case launchd.StateLoaded:
		return 4
	case launchd.StateStopped:
		return 5
	case launchd.StateProtected:
		return 6
	default:
		return 7
	}
}

// snapCursor re-anchors the cursor to the row whose label matches
// m.selectedLabel. Falls back to clamping the previous index if the label is
// no longer visible (e.g. filter excludes it).
func (m *Model) snapCursor() {
	if len(m.filtered) == 0 {
		m.cursor = 0
		m.selectedLabel = ""
		return
	}
	if m.selectedLabel != "" {
		for i, idx := range m.filtered {
			if m.jobs[idx].Label == m.selectedLabel {
				m.cursor = i
				return
			}
		}
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.selectedLabel = m.jobs[m.filtered[m.cursor]].Label
}

// captureSelection records the label under the cursor so the next
// rebuildList can re-snap to it.
func (m *Model) captureSelection() {
	if len(m.filtered) == 0 {
		m.selectedLabel = ""
		return
	}
	if m.cursor < 0 || m.cursor >= len(m.filtered) {
		return
	}
	m.selectedLabel = m.jobs[m.filtered[m.cursor]].Label
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
	Label   string
	Action  string
	Preview string
	Err     error
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

func runActionCmd(action string, job launchd.Job, preview string) tea.Cmd {
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
		return actionMsg{Label: job.Label, Action: action, Preview: preview, Err: err}
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
