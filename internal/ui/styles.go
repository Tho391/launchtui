package ui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/thonq/launchtui/internal/launchd"
)

// Theme is a single named bundle of lipgloss styles for every UI surface.
// All themes share the exact same field set so view.go can be written
// against the abstract Theme without branching per palette. Pressing the
// Theme key (see keys.go) cycles m.themeIdx through allThemes.
type Theme struct {
	Name string

	Title    lipgloss.Style // header title, modal headers, help title
	Subtle   lipgloss.Style // muted text: counts, footer, hints, separators
	Selected lipgloss.Style // selected row marker + label
	Key      lipgloss.Style // accent for keybinding labels + detail field names
	HelpDesc lipgloss.Style // help-overlay descriptions
	Border   lipgloss.Style // list / detail pane borders
	Modal    lipgloss.Style // confirmation-modal border + padding
	Flash    lipgloss.Style // transient flash / info message
	LogLine  lipgloss.Style // body text inside the log-follow pane

	// State carries the per-state colour + weight applied to both the
	// list-row badge glyph and the detail-pane "State" value. Weight is
	// used where it carries meaning (crashed/throttled bold; running
	// bold green; protected/loaded/stopped flat).
	State map[launchd.State]lipgloss.Style
}

// badgeGlyph is the single rune painted in front of every list row. The
// glyph itself is theme-independent; only its colour and weight change
// across themes.
func badgeGlyph(s launchd.State) string {
	switch s {
	case launchd.StateRunning:
		return "●"
	case launchd.StateLoaded:
		return "○"
	case launchd.StateStopped:
		return "·"
	case launchd.StateCrashed:
		return "✖"
	case launchd.StateThrottled:
		return "⚠"
	case launchd.StateProtected:
		return "◆"
	case launchd.StateScheduled:
		return "◷"
	default:
		return "?"
	}
}

// Badge returns the coloured glyph for the row gutter.
func (t *Theme) Badge(s launchd.State) string {
	style, ok := t.State[s]
	if !ok {
		style = t.Subtle
	}
	return style.Render(badgeGlyph(s))
}

// StateText returns the human-readable state label coloured per theme.
func (t *Theme) StateText(s launchd.State) string {
	style, ok := t.State[s]
	if !ok {
		style = t.Subtle
	}
	return style.Render(s.String())
}

// allThemes is the user-visible cycle order. Index 0 is the default.
var allThemes = []*Theme{
	auroraTheme(),
	mochaTheme(),
	highContrastTheme(),
}

// defaultTheme returns the theme NewModel initialises with.
func defaultTheme() *Theme { return allThemes[0] }

// ----- Aurora ----------------------------------------------------------------
// Polished modern dark: violet accent on a midnight backdrop, with vibrant
// state colours pulled from the GitHub Dark palette family.
func auroraTheme() *Theme {
	accent := lipgloss.Color("#7C5CFF") // violet
	muted := lipgloss.Color("#8B949E")  // slate-grey
	text := lipgloss.Color("#E6EDF3")   // off-white
	dim := lipgloss.Color("#6E7681")    // graphite
	success := lipgloss.Color("#3FB950")
	warn := lipgloss.Color("#E3B341")
	danger := lipgloss.Color("#F85149")
	info := lipgloss.Color("#58A6FF")
	cool := lipgloss.Color("#79C0FF")

	return &Theme{
		Name:     "Aurora",
		Title:    lipgloss.NewStyle().Bold(true).Foreground(accent),
		Subtle:   lipgloss.NewStyle().Foreground(muted),
		Selected: lipgloss.NewStyle().Bold(true).Foreground(accent),
		Key:      lipgloss.NewStyle().Bold(true).Foreground(accent),
		HelpDesc: lipgloss.NewStyle().Foreground(muted),
		Border:   lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(dim),
		Modal:    lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accent).Padding(1, 2),
		Flash:    lipgloss.NewStyle().Foreground(warn),
		LogLine:  lipgloss.NewStyle().Foreground(text),
		State: map[launchd.State]lipgloss.Style{
			launchd.StateRunning:   lipgloss.NewStyle().Bold(true).Foreground(success),
			launchd.StateLoaded:    lipgloss.NewStyle().Foreground(cool),
			launchd.StateStopped:   lipgloss.NewStyle().Foreground(dim),
			launchd.StateCrashed:   lipgloss.NewStyle().Bold(true).Foreground(danger),
			launchd.StateThrottled: lipgloss.NewStyle().Bold(true).Foreground(warn),
			launchd.StateProtected: lipgloss.NewStyle().Foreground(info),
			launchd.StateScheduled: lipgloss.NewStyle().Foreground(cool),
			launchd.StateUnknown:   lipgloss.NewStyle().Foreground(muted),
		},
	}
}

// ----- Mocha -----------------------------------------------------------------
// Warm dark pastels on an espresso surface — soft mauve accent, teal keys,
// rose for failures. Inspired by the Catppuccin Mocha family but uses its
// own descriptive name to avoid trademark conflation.
func mochaTheme() *Theme {
	accent := lipgloss.Color("#CBA6F7")  // mauve
	muted := lipgloss.Color("#9399B2")   // overlay-light
	text := lipgloss.Color("#CDD6F4")    // soft white
	dim := lipgloss.Color("#6C7086")     // overlay
	success := lipgloss.Color("#A6E3A1") // green
	warn := lipgloss.Color("#F9E2AF")    // amber
	danger := lipgloss.Color("#F38BA8")  // rose
	info := lipgloss.Color("#89B4FA")    // sapphire
	teal := lipgloss.Color("#94E2D5")    // teal

	return &Theme{
		Name:     "Mocha",
		Title:    lipgloss.NewStyle().Bold(true).Foreground(accent),
		Subtle:   lipgloss.NewStyle().Foreground(muted),
		Selected: lipgloss.NewStyle().Bold(true).Foreground(accent),
		Key:      lipgloss.NewStyle().Bold(true).Foreground(teal),
		HelpDesc: lipgloss.NewStyle().Foreground(muted),
		Border:   lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(dim),
		Modal:    lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accent).Padding(1, 2),
		Flash:    lipgloss.NewStyle().Foreground(warn),
		LogLine:  lipgloss.NewStyle().Foreground(text),
		State: map[launchd.State]lipgloss.Style{
			launchd.StateRunning:   lipgloss.NewStyle().Bold(true).Foreground(success),
			launchd.StateLoaded:    lipgloss.NewStyle().Foreground(info),
			launchd.StateStopped:   lipgloss.NewStyle().Foreground(dim),
			launchd.StateCrashed:   lipgloss.NewStyle().Bold(true).Foreground(danger),
			launchd.StateThrottled: lipgloss.NewStyle().Bold(true).Foreground(warn),
			launchd.StateProtected: lipgloss.NewStyle().Foreground(info),
			launchd.StateScheduled: lipgloss.NewStyle().Foreground(teal),
			launchd.StateUnknown:   lipgloss.NewStyle().Foreground(muted),
		},
	}
}

// ----- High-Contrast ---------------------------------------------------------
// Pure-colour accessibility theme: white text, primary RGB/CMY for states,
// bold weight on everything that communicates status. No muted greys for
// status text — only the chrome (footer, hints) uses a softer tone.
func highContrastTheme() *Theme {
	accent := lipgloss.Color("#00FFFF") // cyan
	text := lipgloss.Color("#FFFFFF")   // pure white
	muted := lipgloss.Color("#C0C0C0")  // silver — still readable on black
	success := lipgloss.Color("#00FF00")
	warn := lipgloss.Color("#FFFF00")
	danger := lipgloss.Color("#FF0000")
	info := lipgloss.Color("#00BFFF")

	return &Theme{
		Name:     "High-Contrast",
		Title:    lipgloss.NewStyle().Bold(true).Foreground(accent),
		Subtle:   lipgloss.NewStyle().Foreground(muted),
		Selected: lipgloss.NewStyle().Bold(true).Foreground(accent),
		Key:      lipgloss.NewStyle().Bold(true).Foreground(accent),
		HelpDesc: lipgloss.NewStyle().Foreground(text),
		Border:   lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(text),
		Modal:    lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accent).Padding(1, 2),
		Flash:    lipgloss.NewStyle().Bold(true).Foreground(warn),
		LogLine:  lipgloss.NewStyle().Foreground(text),
		State: map[launchd.State]lipgloss.Style{
			launchd.StateRunning:   lipgloss.NewStyle().Bold(true).Foreground(success),
			launchd.StateLoaded:    lipgloss.NewStyle().Bold(true).Foreground(info),
			launchd.StateStopped:   lipgloss.NewStyle().Foreground(muted),
			launchd.StateCrashed:   lipgloss.NewStyle().Bold(true).Foreground(danger),
			launchd.StateThrottled: lipgloss.NewStyle().Bold(true).Foreground(warn),
			launchd.StateProtected: lipgloss.NewStyle().Bold(true).Foreground(info),
			launchd.StateScheduled: lipgloss.NewStyle().Bold(true).Foreground(accent),
			launchd.StateUnknown:   lipgloss.NewStyle().Foreground(muted),
		},
	}
}
