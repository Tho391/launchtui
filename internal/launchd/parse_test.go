package launchd

import (
	"strings"
	"testing"
)

const samplePrintOutput = `com.example.foo = {
	active count = 1
	path = /Users/u/Library/LaunchAgents/com.example.foo.plist
	type = LaunchAgent
	state = running
	domain = gui/501 [100008]
	pid = 12345
	program = /usr/local/bin/foo
	last exit code = 0
	run interval = 30
}
`

const samplePrintCrashed = `com.example.bar = {
	active count = 0
	state = not running
	last exit code = 78
	program = /usr/local/bin/bar
}
`

const samplePrintThrottled = `com.example.baz = {
	active count = 0
	state = waiting
	last exit code = 127
	program = /missing/bin
	spawn rate limited
}
`

func TestParsePrintOutput_Running(t *testing.T) {
	st := ParsePrintOutput("com.example.foo", samplePrintOutput)
	if st.State != StateRunning {
		t.Fatalf("want StateRunning, got %v", st.State)
	}
	if st.PID != 12345 {
		t.Fatalf("want pid 12345, got %d", st.PID)
	}
	if st.LastExitCode != 0 {
		t.Fatalf("want exit 0, got %d", st.LastExitCode)
	}
}

func TestParsePrintOutput_Crashed(t *testing.T) {
	st := ParsePrintOutput("com.example.bar", samplePrintCrashed)
	if st.State != StateCrashed {
		t.Fatalf("want StateCrashed, got %v", st.State)
	}
	if st.LastExitCode != 78 {
		t.Fatalf("want exit 78, got %d", st.LastExitCode)
	}
}

func TestParsePrintOutput_Throttled(t *testing.T) {
	st := ParsePrintOutput("com.example.baz", samplePrintThrottled)
	if st.State != StateThrottled {
		t.Fatalf("want StateThrottled, got %v", st.State)
	}
	if !strings.Contains(strings.ToLower(st.Message), "spawn") {
		t.Fatalf("want message to mention spawn, got %q", st.Message)
	}
}

const sampleListOutput = `PID	Status	Label
1234	0	com.example.foo
-	0	com.example.bar
-	78	com.example.crashy
`

func TestParseListOutput(t *testing.T) {
	all := ParseListOutput(sampleListOutput)
	if len(all) != 3 {
		t.Fatalf("want 3 rows, got %d", len(all))
	}
	if all["com.example.foo"].State != StateRunning || all["com.example.foo"].PID != 1234 {
		t.Fatalf("foo wrong: %+v", all["com.example.foo"])
	}
	if all["com.example.bar"].State != StateLoaded {
		t.Fatalf("bar should be loaded: %+v", all["com.example.bar"])
	}
	if all["com.example.crashy"].State != StateCrashed || all["com.example.crashy"].LastExitCode != 78 {
		t.Fatalf("crashy should be crashed/78: %+v", all["com.example.crashy"])
	}
}

const samplePlistXML = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.example.foo</string>
	<key>ProgramArguments</key>
	<array>
		<string>/usr/local/bin/foo</string>
		<string>--flag</string>
	</array>
	<key>StandardOutPath</key>
	<string>/tmp/foo.out</string>
	<key>StandardErrorPath</key>
	<string>/tmp/foo.err</string>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
</dict>
</plist>
`

func TestParsePlistBytes(t *testing.T) {
	job, err := ParsePlistBytes([]byte(samplePlistXML), "/tmp/com.example.foo.plist")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if job.Label != "com.example.foo" {
		t.Fatalf("label: %q", job.Label)
	}
	if job.Program != "/usr/local/bin/foo" {
		t.Fatalf("program: %q", job.Program)
	}
	if len(job.ProgramArgs) != 2 || job.ProgramArgs[1] != "--flag" {
		t.Fatalf("args: %v", job.ProgramArgs)
	}
	if job.StdoutPath != "/tmp/foo.out" || job.StderrPath != "/tmp/foo.err" {
		t.Fatalf("log paths: %q / %q", job.StdoutPath, job.StderrPath)
	}
	if !job.RunAtLoad || !job.KeepAlive {
		t.Fatalf("flags lost: %+v", job)
	}
}
