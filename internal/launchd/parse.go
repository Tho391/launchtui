package launchd

import (
	"bufio"
	"regexp"
	"strconv"
	"strings"
)

// ParsePrintOutput extracts the fields we care about from `launchctl print
// gui/<uid>/<label>` output. It is intentionally lenient: any keys we don't
// recognise are ignored, missing keys leave zero values, and the function
// never returns an error. The state we infer follows these rules:
//
//   - state = running                     → StateRunning
//   - state = waiting/idle/loaded/...     → StateLoaded
//   - last exit code != 0                 → StateCrashed
//   - "spawn rate" / "throttled" hint     → StateThrottled
//   - nothing matched                     → StateUnknown
//
// The label argument is used only to populate the result; if the input
// already contains the label that is fine — we trust the caller.
func ParsePrintOutput(label, out string) JobStatus {
	st := JobStatus{Label: label, State: StateUnknown, LastExitCode: 0, PID: 0}

	reKV := regexp.MustCompile(`^\s*([a-zA-Z][a-zA-Z0-9 _\-]*?)\s*=\s*(.+?)\s*$`)
	scanner := bufio.NewScanner(strings.NewReader(out))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var rawState string
	for scanner.Scan() {
		line := scanner.Text()
		m := reKV.FindStringSubmatch(line)
		if m == nil {
			lower := strings.ToLower(line)
			if strings.Contains(lower, "spawn rate") || strings.Contains(lower, "throttl") {
				st.State = StateThrottled
				if st.Message == "" {
					st.Message = strings.TrimSpace(line)
				}
			}
			continue
		}
		key := strings.ToLower(strings.TrimSpace(m[1]))
		val := strings.TrimSpace(strings.TrimSuffix(m[2], ";"))
		switch key {
		case "state":
			rawState = strings.ToLower(val)
		case "pid":
			if n, err := strconv.Atoi(val); err == nil {
				st.PID = n
			}
		case "last exit code":
			if n, err := strconv.Atoi(val); err == nil {
				st.LastExitCode = n
			}
		case "last exit reason":
			if st.Message == "" {
				st.Message = val
			}
		}
	}

	// Decide a final state. Throttle wins; otherwise running > crashed > loaded.
	if st.State == StateThrottled {
		return st
	}
	switch rawState {
	case "running":
		st.State = StateRunning
	case "":
		// no state line at all — leave Unknown unless other fields suggest more
		if st.PID > 0 {
			st.State = StateRunning
		}
	default:
		st.State = StateLoaded
	}
	if st.LastExitCode != 0 && st.State != StateRunning {
		st.State = StateCrashed
	}
	return st
}

// ParseListLine parses a single line of `launchctl list` (legacy) output:
//
//	PID   Status  Label
//	1234  0       com.example.foo
//	-     0       com.example.bar
//	-     -9      com.example.baz
//
// Returns ok=false for the header row or unparseable input. ok=true for any
// row we managed to split into the three expected columns.
func ParseListLine(line string) (JobStatus, bool) {
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return JobStatus{}, false
	}
	// Header row.
	if fields[0] == "PID" && fields[1] == "Status" {
		return JobStatus{}, false
	}
	st := JobStatus{Label: fields[2], State: StateLoaded}
	if fields[0] != "-" {
		if pid, err := strconv.Atoi(fields[0]); err == nil {
			st.PID = pid
			st.State = StateRunning
		}
	}
	if fields[1] != "-" {
		if code, err := strconv.Atoi(fields[1]); err == nil {
			st.LastExitCode = code
			if code != 0 && st.State != StateRunning {
				st.State = StateCrashed
			}
		}
	}
	return st, true
}

// ParseListOutput walks the whole output of `launchctl list` and returns
// every successfully parsed row, keyed by label.
func ParseListOutput(out string) map[string]JobStatus {
	result := make(map[string]JobStatus)
	scanner := bufio.NewScanner(strings.NewReader(out))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		st, ok := ParseListLine(scanner.Text())
		if !ok {
			continue
		}
		result[st.Label] = st
	}
	return result
}
