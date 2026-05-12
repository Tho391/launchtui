package ui

import "github.com/charmbracelet/bubbles/key"

// keyMap collects every binding into one struct so view.go can render the
// help line without diverging from the actual handlers in update.go.
type keyMap struct {
	Up      key.Binding
	Down    key.Binding
	Top     key.Binding
	Bottom  key.Binding
	PageUp  key.Binding
	PageDn  key.Binding
	Filter  key.Binding
	Start   key.Binding
	Stop    key.Binding
	Restart key.Binding
	Load    key.Binding
	Unload  key.Binding
	Log     key.Binding
	Refresh key.Binding
	Help    key.Binding
	Quit    key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("k/↑", "up")),
		Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("j/↓", "down")),
		Top:     key.NewBinding(key.WithKeys("g", "home"), key.WithHelp("g", "top")),
		Bottom:  key.NewBinding(key.WithKeys("G", "end"), key.WithHelp("G", "bottom")),
		PageUp:  key.NewBinding(key.WithKeys("ctrl+u", "pgup"), key.WithHelp("^U", "page up")),
		PageDn:  key.NewBinding(key.WithKeys("ctrl+d", "pgdown"), key.WithHelp("^D", "page down")),
		Filter:  key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Start:   key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "start")),
		Stop:    key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "stop")),
		Restart: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "restart")),
		Load:    key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "load")),
		Unload:  key.NewBinding(key.WithKeys("U"), key.WithHelp("U", "unload")),
		Log:     key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "log")),
		Refresh: key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "refresh")),
		Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}
