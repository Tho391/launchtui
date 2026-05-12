package ui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/thonq/launchtui/internal/launchd"
)

// Palette is a small named-colour map kept in one place so we can swap themes
// later without touching every view.
var (
	colorAccent    = lipgloss.Color("#7C5CFF")
	colorDim       = lipgloss.Color("241")
	colorRunning   = lipgloss.Color("#3FB950")
	colorLoaded    = lipgloss.Color("244")
	colorStopped   = lipgloss.Color("244")
	colorCrashed   = lipgloss.Color("#F85149")
	colorThrottled = lipgloss.Color("#E3B341")
	colorProtected = lipgloss.Color("#58A6FF")
	colorWarn      = lipgloss.Color("#E3B341")
)

var (
	styleTitle    = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	styleSubtle   = lipgloss.NewStyle().Foreground(colorDim)
	styleSelected = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	styleStatus   = lipgloss.NewStyle().Bold(true)
	styleKey      = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	styleHelpDesc = lipgloss.NewStyle().Foreground(colorDim)
	styleBorder   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorDim)
)

// renderBadge returns a single-rune coloured glyph + short label suitable for
// a list row.
func renderBadge(s launchd.State) string {
	switch s {
	case launchd.StateRunning:
		return lipgloss.NewStyle().Foreground(colorRunning).Render("●")
	case launchd.StateLoaded:
		return lipgloss.NewStyle().Foreground(colorLoaded).Render("○")
	case launchd.StateStopped:
		return lipgloss.NewStyle().Foreground(colorStopped).Render("·")
	case launchd.StateCrashed:
		return lipgloss.NewStyle().Foreground(colorCrashed).Render("✖")
	case launchd.StateThrottled:
		return lipgloss.NewStyle().Foreground(colorThrottled).Render("⚠")
	case launchd.StateProtected:
		return lipgloss.NewStyle().Foreground(colorProtected).Render("◆")
	default:
		return lipgloss.NewStyle().Foreground(colorDim).Render("?")
	}
}

func renderStateText(s launchd.State) string {
	switch s {
	case launchd.StateRunning:
		return lipgloss.NewStyle().Foreground(colorRunning).Render(s.String())
	case launchd.StateCrashed:
		return lipgloss.NewStyle().Foreground(colorCrashed).Render(s.String())
	case launchd.StateThrottled:
		return lipgloss.NewStyle().Foreground(colorThrottled).Render(s.String())
	case launchd.StateProtected:
		return lipgloss.NewStyle().Foreground(colorProtected).Render(s.String())
	default:
		return lipgloss.NewStyle().Foreground(colorDim).Render(s.String())
	}
}
