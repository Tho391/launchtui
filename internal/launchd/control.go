package launchd

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// ErrProtected is returned from any control operation invoked against a
// daemon domain. The TUI surfaces this as an inline status message rather
// than escalating to sudo automatically — that is a v0.2 feature.
var ErrProtected = errors.New("launchtui: this domain requires sudo (run `sudo launchctl …` manually)")

// PreviewCommand returns the argv the TUI would invoke for the given action.
// It is used by the confirmation modal so the user sees the literal
// /bin/launchctl invocation before it runs. Load and Unload fall back to a
// legacy form at runtime; the preview shows the modern (bootstrap / bootout)
// form because that's what we try first.
//
// Returns nil for an unknown action or a job whose domain has no service
// target. Protected (system) daemons still get a preview — the action will
// fail with ErrProtected, but echoing the command we wanted to run is
// useful as a hint for the manual `sudo launchctl …` workflow.
func PreviewCommand(action string, job Job) []string {
	target, hasTarget := job.Domain.ServiceTarget(job.Label)
	domain, hasDomain := job.Domain.DomainTarget()
	switch action {
	case "start":
		if !hasTarget {
			return nil
		}
		return []string{"launchctl", "kickstart", target}
	case "stop":
		if !hasTarget {
			return nil
		}
		return []string{"launchctl", "kill", "TERM", target}
	case "restart":
		if !hasTarget {
			return nil
		}
		return []string{"launchctl", "kickstart", "-k", target}
	case "load":
		if !hasDomain {
			return nil
		}
		return []string{"launchctl", "bootstrap", domain, job.PlistPath}
	case "unload":
		if !hasTarget {
			return nil
		}
		return []string{"launchctl", "bootout", target}
	default:
		return nil
	}
}

// PreviewCommandString joins PreviewCommand with single spaces for display.
// Returns an empty string when PreviewCommand returns nil.
func PreviewCommandString(action string, job Job) string {
	argv := PreviewCommand(action, job)
	if argv == nil {
		return ""
	}
	return strings.Join(argv, " ")
}

// RevealInFinder opens Finder with the given path selected, via `open -R`.
// The context bounds the shell-out per AGENTS.md's 5-second rule.
func RevealInFinder(ctx context.Context, path string) error {
	cmd := exec.CommandContext(ctx, "/usr/bin/open", "-R", path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("open -R %s: %w (%s)", path, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Start asks launchd to fire the job once. For services that have already
// been bootstrapped this is equivalent to `launchctl kickstart`. Bootstrapped
// or not, we try kickstart first because it works whether or not the job is
// currently loaded.
func Start(ctx context.Context, job Job) error {
	if job.Domain.Protected() {
		return ErrProtected
	}
	target, ok := job.Domain.ServiceTarget(job.Label)
	if !ok {
		return fmt.Errorf("no service target for %s", job.Label)
	}
	if _, err := runLaunchctl(ctx, "kickstart", target); err != nil {
		return fmt.Errorf("kickstart %s: %w", job.Label, err)
	}
	return nil
}

// Stop sends SIGTERM (via launchd) to a running instance. It is a no-op if
// the job is not running.
func Stop(ctx context.Context, job Job) error {
	if job.Domain.Protected() {
		return ErrProtected
	}
	target, ok := job.Domain.ServiceTarget(job.Label)
	if !ok {
		return fmt.Errorf("no service target for %s", job.Label)
	}
	if _, err := runLaunchctl(ctx, "kill", "TERM", target); err != nil {
		return fmt.Errorf("kill %s: %w", job.Label, err)
	}
	return nil
}

// Restart combines stop + start with the modern `kickstart -k` flag, which
// kills the current instance (if any) and immediately respawns it.
func Restart(ctx context.Context, job Job) error {
	if job.Domain.Protected() {
		return ErrProtected
	}
	target, ok := job.Domain.ServiceTarget(job.Label)
	if !ok {
		return fmt.Errorf("no service target for %s", job.Label)
	}
	if _, err := runLaunchctl(ctx, "kickstart", "-k", target); err != nil {
		return fmt.Errorf("restart %s: %w", job.Label, err)
	}
	return nil
}

// Load bootstraps a plist into its domain. On older macOS where bootstrap is
// not available we fall back to the legacy `launchctl load`.
func Load(ctx context.Context, job Job) error {
	if job.Domain.Protected() {
		return ErrProtected
	}
	domain, ok := job.Domain.DomainTarget()
	if !ok {
		return fmt.Errorf("no domain target for %s", job.Label)
	}
	if _, err := runLaunchctl(ctx, "bootstrap", domain, job.PlistPath); err == nil {
		return nil
	}
	if _, err := runLaunchctl(ctx, "load", job.PlistPath); err != nil {
		return fmt.Errorf("load %s: %w", job.Label, err)
	}
	return nil
}

// Unload tears down a previously bootstrapped service. On older macOS we fall
// back to `launchctl unload`.
func Unload(ctx context.Context, job Job) error {
	if job.Domain.Protected() {
		return ErrProtected
	}
	target, ok := job.Domain.ServiceTarget(job.Label)
	if !ok {
		return fmt.Errorf("no service target for %s", job.Label)
	}
	if _, err := runLaunchctl(ctx, "bootout", target); err == nil {
		return nil
	}
	if _, err := runLaunchctl(ctx, "unload", job.PlistPath); err != nil {
		return fmt.Errorf("unload %s: %w", job.Label, err)
	}
	return nil
}
