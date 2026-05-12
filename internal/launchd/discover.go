package launchd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"howett.net/plist"
)

// DiscoverDirs returns the four well-known directories launchd reads .plist
// files from, in the order we want to display them. Missing directories are
// silently skipped by Discover.
func DiscoverDirs() []struct {
	Dir    string
	Domain Domain
} {
	home, _ := os.UserHomeDir()
	return []struct {
		Dir    string
		Domain Domain
	}{
		{filepath.Join(home, "Library", "LaunchAgents"), DomainUserAgent},
		{"/Library/LaunchAgents", DomainGlobalAgent},
		{"/Library/LaunchDaemons", DomainGlobalDaemon},
		{"/System/Library/LaunchDaemons", DomainSystemDaemon},
	}
}

// Discover walks every directory returned by DiscoverDirs, parses each
// .plist into a Job, and returns the combined slice sorted by label.
//
// Files we cannot read or parse are skipped — we do not want a single bad
// plist to make the whole list disappear. The returned slice may be empty
// but is never nil.
func Discover() ([]Job, error) {
	var jobs []Job
	for _, src := range DiscoverDirs() {
		found, err := walkDir(src.Dir, src.Domain)
		if err != nil {
			// Skip unreadable directories (e.g. SIP-protected on some setups)
			// but do not abort the entire scan.
			continue
		}
		jobs = append(jobs, found...)
	}
	sort.Slice(jobs, func(i, j int) bool {
		return strings.ToLower(jobs[i].Label) < strings.ToLower(jobs[j].Label)
	})
	return jobs, nil
}

func walkDir(dir string, domain Domain) ([]Job, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}
	var jobs []Job
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".plist") {
			continue
		}
		full := filepath.Join(dir, e.Name())
		job, err := ParsePlistFile(full)
		if err != nil {
			continue
		}
		job.Domain = domain
		if job.Label == "" {
			// Fall back to the filename minus .plist — LaunchControl does the
			// same thing for malformed entries.
			job.Label = strings.TrimSuffix(e.Name(), ".plist")
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

// ParsePlistFile reads a launchd plist file off disk and returns a partially
// populated Job. The Domain field is not set — the caller is responsible for
// that because it depends on the directory the file came from.
func ParsePlistFile(path string) (Job, error) {
	f, err := os.Open(path)
	if err != nil {
		return Job{}, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var raw map[string]any
	dec := plist.NewDecoder(f)
	if err := dec.Decode(&raw); err != nil {
		return Job{}, fmt.Errorf("decode %s: %w", path, err)
	}
	return jobFromPlist(raw, path), nil
}

// ParsePlistBytes is the in-memory counterpart of ParsePlistFile and exists
// mostly so the parse tests can run without touching disk.
func ParsePlistBytes(b []byte, path string) (Job, error) {
	var raw map[string]any
	if _, err := plist.Unmarshal(b, &raw); err != nil {
		return Job{}, fmt.Errorf("decode %s: %w", path, err)
	}
	return jobFromPlist(raw, path), nil
}

func jobFromPlist(raw map[string]any, path string) Job {
	j := Job{PlistPath: path}
	if v, ok := raw["Label"].(string); ok {
		j.Label = v
	}
	if v, ok := raw["Program"].(string); ok {
		j.Program = v
	}
	if args, ok := raw["ProgramArguments"].([]any); ok {
		for _, a := range args {
			if s, ok := a.(string); ok {
				j.ProgramArgs = append(j.ProgramArgs, s)
			}
		}
		if j.Program == "" && len(j.ProgramArgs) > 0 {
			j.Program = j.ProgramArgs[0]
		}
	}
	if v, ok := raw["StandardOutPath"].(string); ok {
		j.StdoutPath = v
	}
	if v, ok := raw["StandardErrorPath"].(string); ok {
		j.StderrPath = v
	}
	if v, ok := raw["Disabled"].(bool); ok {
		j.Disabled = v
	}
	if v, ok := raw["RunAtLoad"].(bool); ok {
		j.RunAtLoad = v
	}
	// KeepAlive may be bool or a dict; we just note whether it is set truthy.
	switch v := raw["KeepAlive"].(type) {
	case bool:
		j.KeepAlive = v
	case map[string]any:
		j.KeepAlive = len(v) > 0
	}
	if n, ok := plistInt(raw["StartInterval"]); ok && n > 0 {
		j.Schedule.Interval = n
	}
	switch v := raw["StartCalendarInterval"].(type) {
	case map[string]any:
		j.Schedule.Calendar = append(j.Schedule.Calendar, parseCalendarEvent(v))
	case []any:
		for _, item := range v {
			if dict, ok := item.(map[string]any); ok {
				j.Schedule.Calendar = append(j.Schedule.Calendar, parseCalendarEvent(dict))
			}
		}
	}
	return j
}

func parseCalendarEvent(dict map[string]any) CalendarEvent {
	e := CalendarEvent{Minute: -1, Hour: -1, Day: -1, Weekday: -1, Month: -1}
	if n, ok := plistInt(dict["Minute"]); ok {
		e.Minute = n
	}
	if n, ok := plistInt(dict["Hour"]); ok {
		e.Hour = n
	}
	if n, ok := plistInt(dict["Day"]); ok {
		e.Day = n
	}
	if n, ok := plistInt(dict["Weekday"]); ok {
		e.Weekday = n
	}
	if n, ok := plistInt(dict["Month"]); ok {
		e.Month = n
	}
	return e
}

// plistInt unwraps the various integer flavours howett.net/plist may decode
// numeric scalars into. Returns ok=false for missing or non-numeric values.
func plistInt(v any) (int, bool) {
	switch x := v.(type) {
	case int:
		return x, true
	case int8:
		return int(x), true
	case int16:
		return int(x), true
	case int32:
		return int(x), true
	case int64:
		return int(x), true
	case uint:
		return int(x), true
	case uint8:
		return int(x), true
	case uint16:
		return int(x), true
	case uint32:
		return int(x), true
	case uint64:
		return int(x), true
	default:
		return 0, false
	}
}
