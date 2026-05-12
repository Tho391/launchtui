// Package launchd is a thin, dependency-light wrapper around the `launchctl`
// CLI plus a parser for launchd plist files.
//
// The package intentionally does not pull in a Cgo XPC binding — shelling out
// is fast enough for an interactive TUI and avoids version drift across macOS
// releases. Every command is bounded by a context timeout supplied by the
// caller (or DefaultTimeout when the caller uses the convenience helpers).
package launchd

import (
	"os/user"
	"strconv"
	"strings"
	"time"
)

// DefaultTimeout caps every shelled-out launchctl invocation.
const DefaultTimeout = 5 * time.Second

// Domain identifies which directory a job's plist lives in. The set of
// directories matches the four well-known locations documented in
// `launchd.plist(5)`.
type Domain int

const (
	DomainUnknown Domain = iota
	DomainUserAgent
	DomainGlobalAgent
	DomainGlobalDaemon
	DomainSystemDaemon
)

// String renders a domain for the UI; the values are short so they fit in a
// fixed-width column.
func (d Domain) String() string {
	switch d {
	case DomainUserAgent:
		return "user"
	case DomainGlobalAgent:
		return "global"
	case DomainGlobalDaemon:
		return "daemon"
	case DomainSystemDaemon:
		return "system"
	default:
		return "?"
	}
}

// Protected reports whether jobs in this domain require sudo to control.
func (d Domain) Protected() bool {
	return d == DomainGlobalDaemon || d == DomainSystemDaemon
}

// ServiceTarget returns the launchctl service target string for a label
// inside this domain — for example "gui/501/com.example.foo" or
// "system/com.apple.cfprefsd.xpc". Returns ok=false if the domain is unknown
// or the current user cannot be resolved.
func (d Domain) ServiceTarget(label string) (string, bool) {
	switch d {
	case DomainUserAgent, DomainGlobalAgent:
		uid, ok := currentUID()
		if !ok {
			return "", false
		}
		return "gui/" + strconv.Itoa(uid) + "/" + label, true
	case DomainGlobalDaemon, DomainSystemDaemon:
		return "system/" + label, true
	default:
		return "", false
	}
}

// DomainTarget returns the launchctl domain target string without a label —
// used for `bootstrap` / `bootout` of a plist file.
func (d Domain) DomainTarget() (string, bool) {
	switch d {
	case DomainUserAgent, DomainGlobalAgent:
		uid, ok := currentUID()
		if !ok {
			return "", false
		}
		return "gui/" + strconv.Itoa(uid), true
	case DomainGlobalDaemon, DomainSystemDaemon:
		return "system", true
	default:
		return "", false
	}
}

// State is the high-level lifecycle bucket we paint a row with.
type State int

const (
	StateUnknown State = iota
	StateRunning
	StateLoaded   // loaded but not currently running
	StateStopped  // not loaded at all
	StateCrashed  // last exit code != 0
	StateThrottled
	StateProtected // belongs to a domain we cannot control without sudo
	// StateScheduled is loaded-with-schedule-but-no-current-PID. Distinguishes
	// recurring jobs (StartInterval / StartCalendarInterval / RunAtLoad)
	// waiting for their next fire from genuinely idle ones.
	StateScheduled
)

func (s State) String() string {
	switch s {
	case StateRunning:
		return "running"
	case StateLoaded:
		return "loaded"
	case StateStopped:
		return "stopped"
	case StateCrashed:
		return "crashed"
	case StateThrottled:
		return "throttled"
	case StateProtected:
		return "protected"
	case StateScheduled:
		return "scheduled"
	default:
		return "unknown"
	}
}

// Job is the union of everything we learnt by reading a .plist file off disk.
// Live state (PID, last exit) lives on JobStatus instead.
type Job struct {
	Label       string
	PlistPath   string
	Domain      Domain
	Program     string
	ProgramArgs []string
	StdoutPath  string
	StderrPath  string
	Disabled    bool
	RunAtLoad   bool
	KeepAlive   bool
	Schedule    Schedule
}

// Schedule captures the firing triggers extracted from a job's plist —
// StartInterval (seconds), StartCalendarInterval (one or many calendar
// events), and RunAtLoad. It is filled in during Discover so the UI can
// label idle-but-recurring jobs as "scheduled" instead of "loaded".
type Schedule struct {
	Interval int             // seconds; 0 means no StartInterval set
	Calendar []CalendarEvent // zero or more StartCalendarInterval entries
}

// CalendarEvent mirrors one StartCalendarInterval dict. Fields are -1 when
// the corresponding key was absent from the plist; an absent field in
// launchd means "match any value", so unset is meaningful.
type CalendarEvent struct {
	Minute  int
	Hour    int
	Day     int
	Weekday int
	Month   int
}

// HasSchedule reports whether the job will be triggered without a manual
// kickstart — i.e. RunAtLoad, a StartInterval, or any StartCalendarInterval.
func (j Job) HasSchedule() bool {
	return j.RunAtLoad || j.Schedule.Interval > 0 || len(j.Schedule.Calendar) > 0
}

// IsAppleSystem reports whether a job is "owned by macOS" for hide-by-default
// purposes — labels in the com.apple.* namespace and any plist that lives
// under /System/Library/LaunchDaemons. Used by both the TUI's A toggle and
// `launchtui list` (which excludes these unless --all is passed).
func (j Job) IsAppleSystem() bool {
	if strings.HasPrefix(strings.ToLower(j.Label), "com.apple.") {
		return true
	}
	if strings.HasPrefix(j.PlistPath, "/System/Library/LaunchDaemons") {
		return true
	}
	return false
}

// JobStatus is the live state of a job as reported by `launchctl print`
// or `launchctl list`.
type JobStatus struct {
	Label        string
	State        State
	PID          int
	LastExitCode int
	Message      string // optional human-friendly note, e.g. "spawn rate limited"
}

// currentUID returns the calling user's numeric UID. It is broken out so that
// tests can mock it via stubbing the variable below.
var currentUID = func() (int, bool) {
	u, err := user.Current()
	if err != nil {
		return 0, false
	}
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return 0, false
	}
	return uid, true
}
