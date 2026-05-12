// launchtui is a terminal UI for inspecting and controlling launchd jobs.
//
// See PLAN.md for the design. macOS-only.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thonq/launchtui/internal/launchd"
	"github.com/thonq/launchtui/internal/ui"
)

const usage = `launchtui — terminal UI for launchd

usage:
  launchtui              start the interactive TUI
  launchtui list [-a]    print every discovered job to stdout
                         (label\tdomain\tstate\tplist_path)
                         -a / --all to include Apple/system jobs
  launchtui --version    print build info and exit
  launchtui -h           this help
`

// Build-time metadata. Overridden via -ldflags -X by goreleaser; the defaults
// keep `go run` / `go build` working without any flags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-v", "--version", "version":
			fmt.Printf("launchtui %s (commit %s, built %s)\n", version, commit, date)
			return
		case "-h", "--help", "help":
			fmt.Print(usage)
			return
		}
	}

	if runtime.GOOS != "darwin" {
		fmt.Fprintln(os.Stderr, "launchtui is macOS-only — launchd does not exist elsewhere.")
		os.Exit(2)
	}

	if len(os.Args) > 1 && os.Args[1] == "list" {
		os.Exit(runList(os.Args[2:]))
	}

	runTUI()
}

func runTUI() {
	model, err := ui.NewModel()
	if err != nil {
		fmt.Fprintf(os.Stderr, "launchtui: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	model.SetProgram(p)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "launchtui: %v\n", err)
		os.Exit(1)
	}
}

func runList(args []string) int {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var includeAll bool
	fs.BoolVar(&includeAll, "all", false, "include Apple/system jobs")
	fs.BoolVar(&includeAll, "a", false, "include Apple/system jobs (shorthand)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: launchtui list [-a|--all]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}

	jobs, err := launchd.Discover()
	if err != nil {
		fmt.Fprintf(os.Stderr, "launchtui list: %v\n", err)
		return 1
	}

	filtered := make([]launchd.Job, 0, len(jobs))
	for _, j := range jobs {
		if !includeAll && j.IsAppleSystem() {
			continue
		}
		filtered = append(filtered, j)
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		return strings.ToLower(filtered[i].Label) < strings.ToLower(filtered[j].Label)
	})

	// Fetch live statuses in parallel. Each Status call has its own
	// context.WithTimeout, so a wedged launchctl never blocks the whole CLI.
	statuses := make([]launchd.JobStatus, len(filtered))
	const workers = 8
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	for i := range filtered {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()
			ctx, cancel := context.WithTimeout(context.Background(), launchd.DefaultTimeout)
			defer cancel()
			st, _ := launchd.Status(ctx, filtered[idx])
			statuses[idx] = st
		}(i)
	}
	wg.Wait()

	w := os.Stdout
	for i, j := range filtered {
		st := statuses[i]
		if st.State == launchd.StateUnknown && j.Domain.Protected() {
			st.State = launchd.StateProtected
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", j.Label, j.Domain, st.State, j.PlistPath)
	}
	return 0
}
