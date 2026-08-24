package app

import (
	tea "charm.land/bubbletea/v2"
	"github.com/Opperiesen/podman-console/internal/config"
	"github.com/Opperiesen/podman-console/internal/podman"
)

// NewProgram wires the persistent configuration, Podman transport and TUI
// model in one place so the executable and tests share the same lifecycle.
func NewProgram(store *config.Store, factory podman.Factory) *tea.Program {
	model := New(store, factory)
	return tea.NewProgram(&model)
}
