package launchd

import (
	"fmt"
	"strings"
)

// FormatSchedule returns a compact human-friendly rendering of a job's
// schedule suitable for a list column or detail row. Empty when the job has
// no scheduled triggers (and is not RunAtLoad-only).
//
// Examples:
//
//	StartInterval=30                              → "30s"
//	StartInterval=300                             → "5min"
//	StartInterval=3600                            → "1h"
//	StartInterval=3900                            → "1h5min"
//	StartCalendarInterval{Weekday:0, Hour:9}      → "Sun 09:00"
//	StartCalendarInterval{Hour:9}                 → "09:00"
//	2 × StartCalendarInterval                     → "Sun 09:00 +1"
//	(only RunAtLoad)                              → "at load"
func FormatSchedule(s Schedule, runAtLoad bool) string {
	var parts []string
	if s.Interval > 0 {
		parts = append(parts, formatInterval(s.Interval))
	}
	if len(s.Calendar) > 0 {
		first := formatCalendarEvent(s.Calendar[0])
		if first == "" {
			first = "cal"
		}
		if len(s.Calendar) == 1 {
			parts = append(parts, first)
		} else {
			parts = append(parts, fmt.Sprintf("%s +%d", first, len(s.Calendar)-1))
		}
	}
	if len(parts) == 0 && runAtLoad {
		return "at load"
	}
	return strings.Join(parts, ", ")
}

// formatInterval renders a seconds value as the largest sensible mixed unit
// (e.g. 3900 → "1h5min", 30 → "30s", 86400 → "1d"). The output is intended
// for compact display, not pretty prose.
func formatInterval(seconds int) string {
	if seconds <= 0 {
		return ""
	}
	const (
		minute = 60
		hour   = 60 * minute
		day    = 24 * hour
	)
	d := seconds / day
	rem := seconds % day
	h := rem / hour
	rem %= hour
	m := rem / minute
	s := rem % minute

	var b strings.Builder
	if d > 0 {
		fmt.Fprintf(&b, "%dd", d)
	}
	if h > 0 {
		fmt.Fprintf(&b, "%dh", h)
	}
	if m > 0 {
		fmt.Fprintf(&b, "%dmin", m)
	}
	if s > 0 && b.Len() == 0 {
		fmt.Fprintf(&b, "%ds", s)
	}
	if b.Len() == 0 {
		return fmt.Sprintf("%ds", seconds)
	}
	return b.String()
}

// formatCalendarEvent renders one StartCalendarInterval entry. Unset fields
// (-1) are omitted; a weekday or day-of-month prefix is followed by an
// "HH:MM" clock if either Hour or Minute is set.
func formatCalendarEvent(e CalendarEvent) string {
	var prefix string
	switch {
	case e.Weekday >= 0:
		prefix = weekdayShort(e.Weekday)
	case e.Day >= 0 && e.Month >= 0:
		prefix = fmt.Sprintf("m%d-d%d", e.Month, e.Day)
	case e.Day >= 0:
		prefix = fmt.Sprintf("day %d", e.Day)
	case e.Month >= 0:
		prefix = fmt.Sprintf("mo %d", e.Month)
	}

	var clock string
	if e.Hour >= 0 || e.Minute >= 0 {
		h := e.Hour
		if h < 0 {
			h = 0
		}
		m := e.Minute
		if m < 0 {
			m = 0
		}
		clock = fmt.Sprintf("%02d:%02d", h, m)
	}

	switch {
	case prefix != "" && clock != "":
		return prefix + " " + clock
	case prefix != "":
		return prefix
	default:
		return clock
	}
}

// weekdayShort maps launchd's 0..7 weekday code to a three-letter name.
// Both 0 and 7 mean Sunday per launchd.plist(5).
func weekdayShort(w int) string {
	names := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	if w < 0 || w > 7 {
		return fmt.Sprintf("wd%d", w)
	}
	return names[w]
}

// ClassifyScheduled returns the State a job should be displayed as once we
// have its live status. Idle jobs (Loaded / Stopped with no PID and a clean
// last exit) that have any scheduling trigger configured become Scheduled;
// everything else passes through unchanged.
func ClassifyScheduled(j Job, st JobStatus) State {
	if st.State != StateLoaded && st.State != StateStopped {
		return st.State
	}
	if st.PID != 0 || st.LastExitCode != 0 {
		return st.State
	}
	if !j.HasSchedule() {
		return st.State
	}
	return StateScheduled
}
