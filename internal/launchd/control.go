package launchd

import (
	"context"
	"errors"
	"fmt"
)

// ErrProtected is returned from any control operation invoked against a
// daemon domain. The TUI surfaces this as an inline status message rather
// than escalating to sudo automatically — that is a v0.2 feature.
var ErrProtected = errors.New("launchtui: this domain requires sudo (run `sudo launchctl …` manually)")

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
