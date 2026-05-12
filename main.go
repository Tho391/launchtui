// launchtui is a terminal UI for inspecting and controlling launchd jobs.
//
// See PLAN.md for the design. macOS-only.
package main

import (
	"fmt"
	"os"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thonq/launchtui/internal/ui"
)

func main() {
	if runtime.GOOS != "darwin" {
		fmt.Fprintln(os.Stderr, "launchtui is macOS-only — launchd does not exist elsewhere.")
		os.Exit(2)
	}

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
