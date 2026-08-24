package ui

import "charm.land/lipgloss/v2"

var (
	Accent = lipgloss.Color("#7D56F4")
	Muted  = lipgloss.Color("#6C7086")
	Text   = lipgloss.Color("#CDD6F4")
	Green  = lipgloss.Color("#A6E3A1")
	Yellow = lipgloss.Color("#F9E2AF")
	Red    = lipgloss.Color("#F38BA8")
	Blue   = lipgloss.Color("#89B4FA")
)

var (
	TitleStyle    = lipgloss.NewStyle().Bold(true).Foreground(Accent)
	TargetStyle   = lipgloss.NewStyle().Foreground(Text)
	MutedStyle    = lipgloss.NewStyle().Foreground(Muted)
	StatusStyle   = lipgloss.NewStyle().Foreground(Green)
	WarningStyle  = lipgloss.NewStyle().Foreground(Yellow)
	ErrorStyle    = lipgloss.NewStyle().Foreground(Red)
	SelectedStyle = lipgloss.NewStyle().Foreground(Text).Background(lipgloss.Color("#313244")).Bold(true)
	HeaderStyle   = lipgloss.NewStyle().Foreground(Blue).Bold(true)
	PanelStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(Muted).Padding(0, 1)
	DialogStyle   = lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).BorderForeground(Yellow).Padding(1, 2)
)
