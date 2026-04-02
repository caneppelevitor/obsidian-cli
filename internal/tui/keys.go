package tui

import "charm.land/bubbles/v2/key"

// KeyMap defines all keybindings for the application.
type KeyMap struct {
	Quit      key.Binding
	Tab       key.Binding
	Submit    key.Binding
	Clear     key.Binding
	Help      key.Binding
	ScrollUp  key.Binding
	ScrollDown key.Binding
}

// DefaultKeyMap returns the default keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "switch tab"),
		),
		Submit: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "submit"),
		),
		Clear: key.NewBinding(
			key.WithKeys("escape"),
			key.WithHelp("esc", "clear"),
		),
		Help: key.NewBinding(
			key.WithKeys("ctrl+?"),
			key.WithHelp("ctrl+?", "help"),
		),
		ScrollUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("pgup", "scroll up"),
		),
		ScrollDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("pgdn", "scroll down"),
		),
	}
}

// ShortHelp returns keybindings for the short help view.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Tab, k.Submit, k.Clear, k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Tab, k.Submit, k.Clear},
		{k.ScrollUp, k.ScrollDown},
		{k.Help, k.Quit},
	}
}
