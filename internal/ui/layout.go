package ui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
	"github.com/Opperiesen/podman-console/internal/domain"
)

const (
	ScreenInventory = "inventory"
	ScreenDetails   = "details"
	ScreenLogs      = "logs"
	ScreenStats     = "stats"
	ModeNormal      = "normal"
	ModeProfiles    = "profiles"
	ModeProfileForm = "profile_form"
	ModeConfirm     = "confirm"
)

type ViewData struct {
	Width, Height   int
	Screen, Mode    string
	Profile         domain.ConnectionProfile
	Connected       bool
	Profiles        []domain.ConnectionProfile
	ActiveProfile   string
	ProfileCursor   int
	ProfileFields   []string
	ProfileFocus    int
	Containers      []domain.ContainerSummary
	Selected        int
	Filter          string
	FilterEditing   bool
	Loading         bool
	Error           error
	Status          string
	Details         *domain.ContainerDetails
	LogContent      string
	LogFollow       bool
	StreamStopped   bool
	Stats           *domain.ContainerStats
	ConfirmAction   string
	ConfirmTarget   string
	ConfirmTargetID string
	FormError       error
	Help            help.Model
	Keys            KeyMap
}

func Render(data ViewData) string {
	width := data.Width
	if width < 1 {
		width = 100
	}
	height := data.Height
	if height < 1 {
		height = 24
	}
	header := TargetHeader(data.Profile, data.Connected)
	body := renderBody(data, width, height)
	status := StatusLine(data.Status, data.Error)
	helpView := renderHelp(data)
	parts := []string{header, body}
	if status != "" {
		parts = append(parts, status)
	}
	if helpView != "" {
		parts = append(parts, helpView)
	}
	content := lipgloss.JoinVertical(lipgloss.Left, parts...)
	if data.Mode == ModeConfirm {
		dialog := Confirmation(data.Profile.DisplayName(), data.ConfirmAction, data.ConfirmTarget, data.ConfirmTargetID)
		content = lipgloss.JoinVertical(lipgloss.Left, content, "", dialog)
	}
	if data.Mode == ModeProfileForm {
		form := ProfileForm(data.ProfileFields, data.ProfileFocus, data.FormError)
		content = lipgloss.JoinVertical(lipgloss.Left, content, "", form)
	}
	return content
}

func renderBody(data ViewData, width, height int) string {
	switch data.Mode {
	case ModeProfiles:
		return renderProfiles(data)
	case ModeProfileForm:
		return renderProfiles(data)
	}
	switch data.Screen {
	case ScreenDetails:
		return renderDetails(data)
	case ScreenLogs:
		return renderLogs(data, width, height)
	case ScreenStats:
		return renderStats(data)
	default:
		return renderInventory(data)
	}
}

func renderInventory(data ViewData) string {
	lines := []string{TitleStyle.Render("PODMAN CONSOLE / CONTENEURS")}
	filter := data.Filter
	if data.FilterEditing {
		filter = filter + "▌"
	}
	if filter != "" {
		lines = append(lines, MutedStyle.Render("Filtre: ")+filter)
	}
	if data.Loading {
		lines = append(lines, WarningStyle.Render("Chargement de l'inventaire…"))
	}
	if len(data.Containers) == 0 && !data.Loading {
		lines = append(lines, EmptyState(data.Filter))
		return PanelStyle.Render(strings.Join(lines, "\n"))
	}
	lines = append(lines, ContainerHeader())
	for i, container := range data.Containers {
		lines = append(lines, ContainerRow(container, i == data.Selected))
	}
	return PanelStyle.Render(strings.Join(lines, "\n"))
}

func renderDetails(data ViewData) string {
	name := "conteneur"
	if data.Details != nil && data.Details.Name != "" {
		name = data.Details.Name
	}
	return PanelStyle.Render(strings.Join([]string{
		TitleStyle.Render("DÉTAILS / " + name),
		DetailView(data.Details),
	}, "\n"))
}

func renderLogs(data ViewData, width, height int) string {
	state := StatusStyle.Render("flux actif")
	if data.StreamStopped {
		state = WarningStyle.Render("flux arrêté — données reçues conservées")
	}
	follow := "pause"
	if !data.LogFollow {
		follow = "reprendre"
	}
	content := data.LogContent
	if content == "" {
		content = MutedStyle.Render("Aucune ligne reçue.")
	}
	maxLines := height - 8
	if maxLines < 4 {
		maxLines = 4
	}
	lines := strings.Split(content, "\n")
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	return PanelStyle.Width(max(20, width-4)).Render(strings.Join([]string{
		TitleStyle.Render("LOGS"),
		fmt.Sprintf("%s  ·  f %s", state, follow),
		strings.Join(lines, "\n"),
	}, "\n"))
}

func renderStats(data ViewData) string {
	return PanelStyle.Render(strings.Join([]string{
		TitleStyle.Render("MÉTRIQUES"),
		StatsView(data.Stats, data.StreamStopped),
	}, "\n"))
}

func renderProfiles(data ViewData) string {
	return PanelStyle.Render(strings.Join([]string{
		TitleStyle.Render("PROFILS PODMAN"),
		ProfileList(data.Profiles, data.ActiveProfile, data.ProfileCursor),
	}, "\n"))
}

func renderHelp(data ViewData) string {
	if data.Help.Width() == 0 {
		data.Help.SetWidth(data.Width)
	}
	var bindings []key.Binding
	switch {
	case data.Mode == ModeProfiles:
		bindings = []key.Binding{data.Keys.Up, data.Keys.Down, data.Keys.Open, data.Keys.New, data.Keys.Edit, data.Keys.Remove, data.Keys.Back, data.Keys.Help, data.Keys.Quit}
	case data.Screen == ScreenDetails:
		bindings = []key.Binding{data.Keys.Start, data.Keys.Stop, data.Keys.Restart, data.Keys.Remove, data.Keys.Logs, data.Keys.Stats, data.Keys.Back, data.Keys.Help, data.Keys.Quit}
	case data.Screen == ScreenLogs || data.Screen == ScreenStats:
		bindings = []key.Binding{data.Keys.Follow, data.Keys.Back, data.Keys.Help, data.Keys.Quit}
	default:
		bindings = []key.Binding{data.Keys.Up, data.Keys.Down, data.Keys.Open, data.Keys.Refresh, data.Keys.Filter, data.Keys.Profiles, data.Keys.Logs, data.Keys.Stats, data.Keys.Help, data.Keys.Quit}
	}
	if data.Help.ShowAll {
		return data.Help.FullHelpView([][]key.Binding{bindings})
	}
	return data.Help.ShortHelpView(bindings)
}
