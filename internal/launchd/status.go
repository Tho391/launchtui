package launchd

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Status returns the live state of a job. It first tries the modern
// `launchctl print <service-target>` command and falls back to scanning
// `launchctl list` for the label if that fails (e.g. on older macOS or for
// jobs we cannot see in our domain).
//
// For protected domains (system daemons) Status still attempts a read-only
// `print`, but returns a JobStatus marked StateProtected if the call is denied
// — the UI displays this as a non-actionable badge.
func Status(ctx context.Context, job Job) (JobStatus, error) {
	target, ok := job.Domain.ServiceTarget(job.Label)
	if !ok {
		return JobStatus{Label: job.Label, State: StateUnknown}, fmt.Errorf("no service target for %s", job.Label)
	}

	out, err := runLaunchctl(ctx, "print", target)
	if err == nil {
		st := ParsePrintOutput(job.Label, out)
		if job.Domain.Protected() && st.State == StateUnknown {
			st.State = StateProtected
		}
		st.State = ClassifyScheduled(job, st)
		return st, nil
	}

	// Fallback: parse `launchctl list` for the label.
	list, listErr := runLaunchctl(ctx, "list")
	if listErr == nil {
		all := ParseListOutput(list)
		if st, found := all[job.Label]; found {
			st.State = ClassifyScheduled(job, st)
			return st, nil
		}
	}

	// Nothing matched. Distinguish "loaded but launchctl says nothing"
	// from a protected daemon.
	if job.Domain.Protected() {
		return JobStatus{Label: job.Label, State: StateProtected, Message: "needs sudo"}, nil
	}
	st := JobStatus{Label: job.Label, State: StateStopped, Message: strings.TrimSpace(out)}
	st.State = ClassifyScheduled(job, st)
	return st, nil
}

// Print returns the raw `launchctl print` output — handy for a "show me the
// details" view in the UI without us re-rendering it.
func Print(ctx context.Context, job Job) (string, error) {
	target, ok := job.Domain.ServiceTarget(job.Label)
	if !ok {
		return "", fmt.Errorf("no service target for %s", job.Label)
	}
	return runLaunchctl(ctx, "print", target)
}

// runLaunchctl shells out to `/bin/launchctl`. It captures stdout+stderr
// together because launchctl is inconsistent about which stream it uses for
// useful output (especially on error). The combined buffer is returned to the
// caller along with any non-zero exit.
func runLaunchctl(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "/bin/launchctl", args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}
