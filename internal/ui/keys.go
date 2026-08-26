package ui

import "charm.land/bubbles/v2/key"

// KeyMap contains the commands that are shared by every screen. Keeping the
// bindings here makes the on-screen help and the model's input handling agree.
type KeyMap struct {
	Quit     key.Binding
	Help     key.Binding
	Refresh  key.Binding
	Profiles key.Binding
	Images   key.Binding
	Pull     key.Binding
	Back     key.Binding
	Confirm  key.Binding
	Cancel   key.Binding
	Up       key.Binding
	Down     key.Binding
	Open     key.Binding
	Filter   key.Binding
	Start    key.Binding
	Stop     key.Binding
	Restart  key.Binding
	Remove   key.Binding
	Details  key.Binding
	Logs     key.Binding
	Stats    key.Binding
	Follow   key.Binding
	Edit     key.Binding
	New      key.Binding
}

func NewKeyMap() KeyMap {
	return KeyMap{
		Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quitter")),
		Help:     key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "aide")),
		Refresh:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "actualiser")),
		Profiles: key.NewBinding(key.WithKeys("c", "p"), key.WithHelp("c", "profils")),
		Images:   key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "images")),
		Pull:     key.NewBinding(key.WithKeys("P"), key.WithHelp("P", "télécharger")),
		Back:     key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "retour")),
		Confirm:  key.NewBinding(key.WithKeys("enter", "y"), key.WithHelp("enter", "confirmer")),
		Cancel:   key.NewBinding(key.WithKeys("esc", "n"), key.WithHelp("esc", "annuler")),
		Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "monter")),
		Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "descendre")),
		Open:     key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "ouvrir")),
		Filter:   key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filtrer")),
		Start:    key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "démarrer")),
		Stop:     key.NewBinding(key.WithKeys("x", "t"), key.WithHelp("x", "arrêter")),
		Restart:  key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "redémarrer")),
		Remove:   key.NewBinding(key.WithKeys("D", "d"), key.WithHelp("D", "supprimer")),
		Details:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "détails")),
		Logs:     key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "logs")),
		Stats:    key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "métriques")),
		Follow:   key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "suivre")),
		Edit:     key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "modifier")),
		New:      key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "nouveau")),
	}
}

func (k KeyMap) InventoryHelp() []key.Binding {
	return []key.Binding{k.Open, k.Refresh, k.Filter, k.Profiles, k.Logs, k.Stats, k.Quit}
}

func (k KeyMap) DetailHelp() []key.Binding {
	return []key.Binding{k.Start, k.Stop, k.Restart, k.Remove, k.Logs, k.Stats, k.Back, k.Quit}
}

func (k KeyMap) StreamHelp() []key.Binding {
	return []key.Binding{k.Follow, k.Back, k.Quit}
}
