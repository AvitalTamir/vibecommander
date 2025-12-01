package app

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the key bindings for the application.
type KeyMap struct {
	// Global keys
	Quit         key.Binding
	Help         key.Binding
	FocusTree    key.Binding
	FocusContent key.Binding
	ToggleMini   key.Binding

	// Navigation
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Home     key.Binding
	End      key.Binding

	// Actions
	Enter  key.Binding
	Back   key.Binding
	Delete key.Binding

	// AI
	LaunchAI key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		// Global
		Quit: key.NewBinding(
			key.WithKeys("ctrl+q"),
			key.WithHelp("ctrl+q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("ctrl+h"),
			key.WithHelp("ctrl+h", "help"),
		),
		FocusTree: key.NewBinding(
			key.WithKeys("alt+1"),
			key.WithHelp("M-1", "focus tree"),
		),
		FocusContent: key.NewBinding(
			key.WithKeys("alt+2"),
			key.WithHelp("M-2", "focus content"),
		),
		ToggleMini: key.NewBinding(
			key.WithKeys("alt+3"),
			key.WithHelp("M-3", "toggle terminal"),
		),

		// Navigation
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "left/collapse"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "right/expand"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+u"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+d"),
			key.WithHelp("pgdown", "page down"),
		),
		Home: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("home/g", "go to top"),
		),
		End: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("end/G", "go to bottom"),
		),

		// Actions
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select/open"),
		),
		Back: key.NewBinding(
			key.WithKeys("backspace", "esc"),
			key.WithHelp("backspace", "back"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d", "delete"),
			key.WithHelp("d", "delete"),
		),

		// AI
		LaunchAI: key.NewBinding(
			key.WithKeys("alt+a"),
			key.WithHelp("M-a", "launch AI"),
		),
	}
}

// ShortHelp returns the short help text for the key map.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit, k.FocusTree, k.Enter}
}

// FullHelp returns the full help text for the key map.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.PageUp, k.PageDown, k.Home, k.End},
		{k.Enter, k.Back, k.Delete},
		{k.FocusTree, k.FocusContent, k.ToggleMini},
		{k.LaunchAI, k.Help, k.Quit},
	}
}
