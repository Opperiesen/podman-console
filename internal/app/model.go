package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/Opperiesen/podman-console/internal/config"
	"github.com/Opperiesen/podman-console/internal/domain"
	"github.com/Opperiesen/podman-console/internal/podman"
	"github.com/Opperiesen/podman-console/internal/ui"
)

type Model struct {
	store   *config.Store
	factory podman.Factory
	ctx     context.Context
	cancel  context.CancelFunc

	file    config.File
	client  podman.Client
	profile domain.ConnectionProfile

	keys   ui.KeyMap
	help   help.Model
	screen string
	mode   string

	width  int
	height int

	containers  []domain.ContainerSummary
	selected    int
	filter      string
	filtering   bool
	filterInput textinput.Model

	loading       bool
	detailLoading bool
	connected     bool
	details       *domain.ContainerDetails

	status string
	err    error

	generation    uint64
	requestCancel context.CancelFunc

	pendingAction domain.Action
	pendingTarget string
	pendingID     string

	profileCursor    int
	profileInputs    []textinput.Model
	profileFocus     int
	editingProfile   string
	profileFormError error

	viewport         viewport.Model
	streamCancel     context.CancelFunc
	streamGeneration uint64
	streamReturn     string
	logLines         []domain.LogLine
	logFollow        bool
	streamStopped    bool
	stats            *domain.ContainerStats
}

func New(store *config.Store, factory podman.Factory) Model {
	ctx, cancel := context.WithCancel(context.Background())
	filter := textinput.New()
	filter.Prompt = ""
	filter.Placeholder = "nom, image ou identifiant"
	filter.CharLimit = 120
	filter.SetWidth(36)

	inputs := make([]textinput.Model, 3)
	placeholders := []string{"nom du profil", "unix:///… ou ssh://…", "chemin vers identity (optionnel)"}
	for i := range inputs {
		inputs[i] = textinput.New()
		inputs[i].Prompt = ""
		inputs[i].Placeholder = placeholders[i]
		inputs[i].CharLimit = 512
		inputs[i].SetWidth(56)
	}

	return Model{
		store:         store,
		factory:       factory,
		ctx:           ctx,
		cancel:        cancel,
		file:          config.Default(),
		keys:          ui.NewKeyMap(),
		help:          help.New(),
		screen:        ui.ScreenInventory,
		mode:          ui.ModeNormal,
		filterInput:   filter,
		profileInputs: inputs,
		viewport:      viewport.New(viewport.WithWidth(80), viewport.WithHeight(12)),
		logFollow:     true,
	}
}

func NewModel(store *config.Store, factory podman.Factory) Model { return New(store, factory) }

func (m Model) Init() tea.Cmd { return loadConfigCmd(m.store) }

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch message := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = message.Width, message.Height
		m.help.SetWidth(message.Width)
		m.resizeViewport()
		return m, nil
	case ConfigLoadedMsg:
		return m.handleConfigLoaded(message)
	case ProfileConnectedMsg:
		if message.Generation != m.generation {
			return m, nil
		}
		m.requestCancel = nil
		m.loading = false
		m.profile = message.Profile
		m.client = message.Client
		m.connected = message.Client != nil
		if message.Err != nil {
			m.err = friendlyError(message.Err)
			m.status = ""
			return m, nil
		}
		m.containers = message.Containers
		m.selected = clamp(m.selected, 0, len(m.visibleContainers())-1)
		m.err = nil
		m.status = fmt.Sprintf("Connecté à %s · %d conteneur(s)", m.profile.DisplayName(), len(m.containers))
		return m, nil
	case InventoryLoadedMsg:
		if message.Generation != m.generation {
			return m, nil
		}
		m.requestCancel = nil
		m.loading = false
		if message.Err != nil {
			m.err = friendlyError(message.Err)
			m.status = ""
			return m, nil
		}
		m.containers = message.Containers
		m.selected = clamp(m.selected, 0, len(m.visibleContainers())-1)
		m.err = nil
		m.status = fmt.Sprintf("Inventaire actualisé · %d conteneur(s)", len(m.containers))
		return m, nil
	case DetailsLoadedMsg:
		if message.Generation != m.generation {
			return m, nil
		}
		m.requestCancel = nil
		m.detailLoading = false
		if message.Err != nil {
			m.err = friendlyError(message.Err)
			return m, nil
		}
		m.details = &message.Details
		m.err = nil
		return m, nil
	case OperationFinishedMsg:
		return m.handleOperation(message)
	case logStreamEvent:
		return m.handleLogEvent(message)
	case statsStreamEvent:
		return m.handleStatsEvent(message)
	case tea.KeyMsg:
		return m.handleKey(message)
	}
	return m, nil
}

func (m *Model) View() tea.View {
	data := ui.ViewData{
		Width:           m.width,
		Height:          m.height,
		Screen:          m.screen,
		Mode:            m.mode,
		Profile:         m.profile,
		Connected:       m.connected,
		Profiles:        m.file.Profiles,
		ActiveProfile:   m.file.Active,
		ProfileCursor:   m.profileCursor,
		ProfileFields:   m.profileFieldValues(),
		ProfileFocus:    m.profileFocus,
		Containers:      m.visibleContainers(),
		Selected:        m.selected,
		Filter:          m.filter,
		FilterEditing:   m.filtering,
		Loading:         m.loading || m.detailLoading,
		Error:           m.err,
		Status:          m.status,
		Details:         m.details,
		LogContent:      m.viewport.View(),
		LogFollow:       m.logFollow,
		StreamStopped:   m.streamStopped,
		Stats:           m.stats,
		ConfirmAction:   actionLabel(m.pendingAction),
		ConfirmTarget:   m.pendingTarget,
		ConfirmTargetID: m.pendingID,
		FormError:       m.profileFormError,
		Help:            m.help,
		Keys:            m.keys,
	}
	v := tea.NewView(ui.Render(data))
	v.AltScreen = true
	v.WindowTitle = "Podman Console"
	return v
}

func (m *Model) handleConfigLoaded(message ConfigLoadedMsg) (tea.Model, tea.Cmd) {
	if message.Err != nil {
		m.err = friendlyError(message.Err)
		m.mode = ui.ModeProfiles
		return m, nil
	}
	m.file = message.File
	if profile, ok := m.file.ActiveProfile(); ok {
		return m, m.beginProfile(profile)
	}
	m.mode = ui.ModeProfiles
	m.profile = domain.ConnectionProfile{}
	m.status = "Configurez une première cible Podman."
	return m, nil
}

func (m *Model) handleOperation(message OperationFinishedMsg) (tea.Model, tea.Cmd) {
	if message.Generation != m.generation {
		return m, nil
	}
	m.requestCancel = nil
	m.loading = false
	if message.Err != nil {
		m.err = friendlyError(message.Err)
		m.status = ""
		if isStaleTarget(message.Err) {
			return m, m.refresh()
		}
		return m, nil
	}
	m.err = nil
	m.status = fmt.Sprintf("%s réussi pour %s · actualisation…", actionLabel(message.Action), shortTarget(message.TargetID))
	if message.Action == domain.ActionRemove {
		m.screen = ui.ScreenInventory
		m.details = nil
	}
	ctx, generation := m.beginRequest()
	m.loading = true
	commands := []tea.Cmd{listContainersCmd(ctx, m.client, generation)}
	if m.screen == ui.ScreenDetails && message.Action != domain.ActionRemove {
		commands = append(commands, inspectContainerCmd(ctx, m.client, message.TargetID, generation))
	}
	return m, tea.Batch(commands...)
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.mode == ui.ModeConfirm {
		if key.Matches(msg, m.keys.Confirm) {
			action, id := m.pendingAction, m.pendingID
			m.mode = ui.ModeNormal
			m.pendingAction, m.pendingID, m.pendingTarget = "", "", ""
			return m, m.runOperation(action, id)
		}
		if key.Matches(msg, m.keys.Cancel) {
			m.mode = ui.ModeNormal
			m.pendingAction, m.pendingID, m.pendingTarget = "", "", ""
			m.status = "Opération annulée ; la cible n’a pas été modifiée."
			return m, nil
		}
		return m, nil
	}
	if m.mode == ui.ModeProfileForm {
		return m.handleProfileFormKey(msg)
	}
	if m.filtering {
		if key.Matches(msg, m.keys.Cancel) {
			m.filtering = false
			m.filterInput.Blur()
			return m, nil
		}
		if key.Matches(msg, m.keys.Confirm) {
			m.filtering = false
			m.filterInput.Blur()
			m.filter = m.filterInput.Value()
			m.selected = 0
			return m, nil
		}
		updated, cmd := m.filterInput.Update(msg)
		m.filterInput = updated
		m.filter = updated.Value()
		m.selected = clamp(m.selected, 0, len(m.visibleContainers())-1)
		return m, cmd
	}
	if m.mode == ui.ModeProfiles {
		return m.handleProfilesKey(msg)
	}
	if key.Matches(msg, m.keys.Quit) {
		m.cancel()
		m.stopStream()
		return m, tea.Quit
	}
	if key.Matches(msg, m.keys.Help) {
		m.help.ShowAll = !m.help.ShowAll
		return m, nil
	}
	if m.screen == ui.ScreenLogs || m.screen == ui.ScreenStats {
		if key.Matches(msg, m.keys.Back) {
			m.stopStream()
			m.screen = m.streamReturn
			m.streamStopped = false
			return m, nil
		}
		if m.screen == ui.ScreenLogs && key.Matches(msg, m.keys.Follow) {
			m.logFollow = !m.logFollow
			if m.logFollow {
				m.viewport.GotoBottom()
			}
			return m, nil
		}
		updated, cmd := m.viewport.Update(msg)
		m.viewport = updated
		return m, cmd
	}

	if key.Matches(msg, m.keys.Profiles) {
		m.mode = ui.ModeProfiles
		m.profileCursor = indexOfProfile(m.file.Profiles, m.file.Active)
		return m, nil
	}
	if key.Matches(msg, m.keys.Refresh) {
		if m.screen == ui.ScreenDetails && m.details != nil && m.client != nil {
			ctx, generation := m.beginRequest()
			m.detailLoading = true
			return m, inspectContainerCmd(ctx, m.client, m.details.ID, generation)
		}
		return m, m.refresh()
	}
	if key.Matches(msg, m.keys.Filter) {
		m.filtering = true
		m.filterInput.SetValue(m.filter)
		return m, m.filterInput.Focus()
	}
	if key.Matches(msg, m.keys.Up) {
		m.selected = clamp(m.selected-1, 0, len(m.visibleContainers())-1)
		return m, nil
	}
	if key.Matches(msg, m.keys.Down) {
		m.selected = clamp(m.selected+1, 0, len(m.visibleContainers())-1)
		return m, nil
	}
	if key.Matches(msg, m.keys.Open) && m.screen == ui.ScreenInventory {
		return m, m.openDetails()
	}
	if key.Matches(msg, m.keys.Start) {
		return m, m.runOperation(domain.ActionStart, m.selectedID())
	}
	if key.Matches(msg, m.keys.Stop) {
		return m, m.requestConfirmation(domain.ActionStop)
	}
	if key.Matches(msg, m.keys.Restart) {
		return m, m.requestConfirmation(domain.ActionRestart)
	}
	if key.Matches(msg, m.keys.Remove) {
		return m, m.requestConfirmation(domain.ActionRemove)
	}
	if key.Matches(msg, m.keys.Logs) {
		return m, m.openLogs()
	}
	if key.Matches(msg, m.keys.Stats) {
		return m, m.openStats()
	}
	if key.Matches(msg, m.keys.Back) && m.screen == ui.ScreenDetails {
		m.screen = ui.ScreenInventory
		m.details = nil
		return m, nil
	}
	return m, nil
}

func (m *Model) handleProfilesKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.Back) {
		m.mode = ui.ModeNormal
		return m, nil
	}
	if key.Matches(msg, m.keys.Quit) {
		m.cancel()
		return m, tea.Quit
	}
	if key.Matches(msg, m.keys.Up) {
		m.profileCursor = clamp(m.profileCursor-1, 0, len(m.file.Profiles)-1)
		return m, nil
	}
	if key.Matches(msg, m.keys.Down) {
		m.profileCursor = clamp(m.profileCursor+1, 0, len(m.file.Profiles)-1)
		return m, nil
	}
	if key.Matches(msg, m.keys.New) {
		m.openProfileForm(nil)
		return m, m.focusProfileInput()
	}
	if key.Matches(msg, m.keys.Edit) && len(m.file.Profiles) > 0 {
		profile := m.file.Profiles[m.profileCursor]
		m.openProfileForm(&profile)
		return m, m.focusProfileInput()
	}
	if key.Matches(msg, m.keys.Remove) && len(m.file.Profiles) > 0 {
		return m, m.removeProfile()
	}
	if key.Matches(msg, m.keys.Open) && len(m.file.Profiles) > 0 {
		return m, m.beginProfile(m.file.Profiles[m.profileCursor])
	}
	return m, nil
}

func (m *Model) handleProfileFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.Cancel) {
		m.mode = ui.ModeProfiles
		m.profileFormError = nil
		m.blurProfileInputs()
		return m, nil
	}
	if msg.String() == "ctrl+s" || key.Matches(msg, m.keys.Confirm) && m.profileFocus == len(m.profileInputs)-1 {
		return m, m.saveProfile()
	}
	if msg.String() == "tab" || msg.String() == "shift+tab" {
		if msg.String() == "shift+tab" {
			m.profileFocus = clamp(m.profileFocus-1, 0, len(m.profileInputs)-1)
		} else {
			m.profileFocus = (m.profileFocus + 1) % len(m.profileInputs)
		}
		return m, m.focusProfileInput()
	}
	updated, cmd := m.profileInputs[m.profileFocus].Update(msg)
	m.profileInputs[m.profileFocus] = updated
	m.profileFormError = updated.Err
	return m, cmd
}

func (m *Model) openProfileForm(profile *domain.ConnectionProfile) {
	m.mode = ui.ModeProfileForm
	m.profileFormError = nil
	m.profileFocus = 0
	m.editingProfile = ""
	values := []string{"", "", ""}
	if profile != nil {
		values = []string{profile.Name, profile.URI, profile.IdentityPath}
		m.editingProfile = profile.Name
	}
	for i, value := range values {
		m.profileInputs[i].SetValue(value)
	}
	m.blurProfileInputs()
}

func (m *Model) focusProfileInput() tea.Cmd {
	m.blurProfileInputs()
	return m.profileInputs[m.profileFocus].Focus()
}

func (m *Model) blurProfileInputs() {
	for i := range m.profileInputs {
		m.profileInputs[i].Blur()
	}
}

func (m *Model) profileFieldValues() []string {
	values := make([]string, len(m.profileInputs))
	for i := range m.profileInputs {
		values[i] = m.profileInputs[i].Value()
	}
	return values
}

func (m *Model) saveProfile() tea.Cmd {
	values := m.profileFieldValues()
	profile := domain.ConnectionProfile{Name: strings.TrimSpace(values[0]), URI: strings.TrimSpace(values[1]), IdentityPath: strings.TrimSpace(values[2])}
	if err := profile.Validate(); err != nil {
		m.profileFormError = err
		return nil
	}
	if m.editingProfile != "" && !strings.EqualFold(m.editingProfile, profile.Name) {
		m.file.Remove(m.editingProfile)
	}
	if err := m.file.Upsert(profile); err != nil {
		m.profileFormError = err
		return nil
	}
	m.file.Active = profile.Name
	if err := m.saveConfig(); err != nil {
		m.profileFormError = err
		return nil
	}
	m.profileCursor = indexOfProfile(m.file.Profiles, profile.Name)
	m.profileFormError = nil
	m.blurProfileInputs()
	return m.beginProfile(profile)
}

func (m *Model) removeProfile() tea.Cmd {
	name := m.file.Profiles[m.profileCursor].Name
	if !m.file.Remove(name) {
		return nil
	}
	if err := m.saveConfig(); err != nil {
		m.err = err
		return nil
	}
	m.profileCursor = clamp(m.profileCursor, 0, len(m.file.Profiles)-1)
	if profile, ok := m.file.ActiveProfile(); ok {
		return m.beginProfile(profile)
	}
	m.client = nil
	m.connected = false
	m.profile = domain.ConnectionProfile{}
	m.mode = ui.ModeProfiles
	m.status = "Profil supprimé ; configurez une nouvelle cible."
	return nil
}

func (m *Model) beginProfile(profile domain.ConnectionProfile) tea.Cmd {
	m.stopStream()
	m.profile = profile
	m.file.Active = profile.Name
	_ = m.saveConfig()
	m.client = nil
	m.connected = false
	m.screen = ui.ScreenInventory
	m.mode = ui.ModeNormal
	m.details = nil
	m.err = nil
	m.loading = true
	m.status = "Connexion à " + profile.DisplayName() + "…"
	ctx, generation := m.beginRequest()
	return connectProfileCmd(ctx, m.factory, profile, generation)
}

func (m *Model) refresh() tea.Cmd {
	if m.client == nil {
		m.err = fmt.Errorf("aucune cible Podman connectée")
		return nil
	}
	ctx, generation := m.beginRequest()
	m.loading = true
	m.status = "Actualisation de l’inventaire…"
	return listContainersCmd(ctx, m.client, generation)
}

func (m *Model) beginRequest() (context.Context, uint64) {
	if m.requestCancel != nil {
		m.requestCancel()
	}
	m.generation++
	ctx, cancel := context.WithCancel(m.ctx)
	m.requestCancel = cancel
	return ctx, m.generation
}

func (m *Model) runOperation(action domain.Action, id string) tea.Cmd {
	if id == "" {
		m.err = fmt.Errorf("aucun conteneur sélectionné")
		return nil
	}
	if m.client == nil {
		m.err = fmt.Errorf("aucune cible Podman connectée")
		return nil
	}
	ctx, generation := m.beginRequest()
	m.loading = true
	m.err = nil
	m.status = actionLabel(action) + "…"
	return operationCmd(ctx, m.client, action, id, generation)
}

func (m *Model) requestConfirmation(action domain.Action) tea.Cmd {
	id := m.selectedID()
	if id == "" {
		m.err = fmt.Errorf("aucun conteneur sélectionné")
		return nil
	}
	m.pendingAction = action
	m.pendingID = id
	m.pendingTarget = m.selectedName()
	m.mode = ui.ModeConfirm
	m.err = nil
	return nil
}

func (m *Model) openDetails() tea.Cmd {
	id := m.selectedID()
	if id == "" || m.client == nil {
		m.err = fmt.Errorf("aucun conteneur sélectionné")
		return nil
	}
	m.screen = ui.ScreenDetails
	m.details = nil
	m.detailLoading = true
	m.err = nil
	ctx, generation := m.beginRequest()
	return inspectContainerCmd(ctx, m.client, id, generation)
}

func (m *Model) openLogs() tea.Cmd {
	id := m.selectedID()
	if id == "" || m.client == nil {
		m.err = fmt.Errorf("aucun conteneur sélectionné")
		return nil
	}
	m.stopStream()
	m.streamGeneration++
	generation := m.streamGeneration
	m.streamReturn = m.screen
	m.screen = ui.ScreenLogs
	m.logLines = nil
	m.logFollow = true
	m.streamStopped = false
	m.viewport.SetContent("")
	ctx, cancel := context.WithCancel(m.ctx)
	m.streamCancel = cancel
	return startLogStreamCmd(ctx, m.client, id, podman.LogOptions{Follow: true, Tail: 200, Timestamps: true}, generation)
}

func (m *Model) openStats() tea.Cmd {
	id := m.selectedID()
	if id == "" || m.client == nil {
		m.err = fmt.Errorf("aucun conteneur sélectionné")
		return nil
	}
	m.stopStream()
	m.streamGeneration++
	generation := m.streamGeneration
	m.streamReturn = m.screen
	m.screen = ui.ScreenStats
	m.stats = nil
	m.streamStopped = false
	ctx, cancel := context.WithCancel(m.ctx)
	m.streamCancel = cancel
	return startStatsStreamCmd(ctx, m.client, id, generation)
}

func (m *Model) stopStream() {
	m.streamGeneration++
	if m.streamCancel != nil {
		m.streamCancel()
		m.streamCancel = nil
	}
}

func (m *Model) handleLogEvent(event logStreamEvent) (tea.Model, tea.Cmd) {
	if event.Generation != m.streamGeneration {
		return m, nil
	}
	if event.Line != nil {
		m.logLines = append(m.logLines, *event.Line)
		m.viewport.SetContent(formatLogs(m.logLines))
		if m.logFollow {
			m.viewport.GotoBottom()
		}
	}
	if event.Done {
		m.streamStopped = true
		m.streamCancel = nil
		if event.Err != nil && !errors.Is(event.Err, context.Canceled) {
			m.err = friendlyError(event.Err)
		}
		return m, nil
	}
	if event.Next != nil {
		return m, waitLogStream(event.Next, event.Generation)
	}
	return m, nil
}

func (m *Model) handleStatsEvent(event statsStreamEvent) (tea.Model, tea.Cmd) {
	if event.Generation != m.streamGeneration {
		return m, nil
	}
	if event.Sample != nil {
		m.stats = event.Sample
		m.err = nil
	}
	if event.Done {
		m.streamStopped = true
		m.streamCancel = nil
		if event.Err != nil && !errors.Is(event.Err, context.Canceled) {
			m.err = friendlyError(event.Err)
		}
		return m, nil
	}
	if event.Next != nil {
		return m, waitStatsStream(event.Next, event.Generation)
	}
	return m, nil
}

func (m *Model) resizeViewport() {
	width := m.width - 4
	if width < 20 {
		width = 20
	}
	height := m.height - 8
	if height < 4 {
		height = 4
	}
	m.viewport.SetWidth(width)
	m.viewport.SetHeight(height)
}

func (m *Model) saveConfig() error {
	if m.store == nil {
		return nil
	}
	return m.store.Save(m.file)
}

func (m *Model) visibleContainers() []domain.ContainerSummary {
	query := strings.ToLower(strings.TrimSpace(m.filter))
	if query == "" {
		return append([]domain.ContainerSummary(nil), m.containers...)
	}
	result := make([]domain.ContainerSummary, 0, len(m.containers))
	for _, container := range m.containers {
		fields := []string{container.Name, container.ID, container.Image, container.Status, container.State.String()}
		matched := false
		for _, field := range fields {
			if strings.Contains(strings.ToLower(field), query) {
				matched = true
				break
			}
		}
		if matched {
			result = append(result, container)
		}
	}
	return result
}

func (m *Model) selectedContainer() (domain.ContainerSummary, bool) {
	containers := m.visibleContainers()
	if m.selected < 0 || m.selected >= len(containers) {
		return domain.ContainerSummary{}, false
	}
	return containers[m.selected], true
}

func (m *Model) selectedID() string {
	if m.screen == ui.ScreenDetails && m.details != nil {
		return m.details.ID
	}
	container, ok := m.selectedContainer()
	if !ok {
		return ""
	}
	return container.ID
}

func (m *Model) selectedName() string {
	if m.screen == ui.ScreenDetails && m.details != nil && m.details.Name != "" {
		return m.details.Name
	}
	container, ok := m.selectedContainer()
	if !ok {
		return "conteneur inconnu"
	}
	if container.Name != "" {
		return container.Name
	}
	return shortTarget(container.ID)
}

func formatLogs(lines []domain.LogLine) string {
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		prefix := line.Stream
		if prefix == "" {
			prefix = "stdout"
		}
		result = append(result, fmt.Sprintf("[%s] %s", prefix, line.Text))
	}
	return strings.Join(result, "\n")
}

func friendlyError(err error) error {
	if err == nil {
		return nil
	}
	var operation *domain.OperationError
	if errors.As(err, &operation) {
		var prefix string
		switch operation.Category {
		case domain.ErrorAuthorization:
			prefix = "Autorisation refusée"
		case domain.ErrorTransport:
			prefix = "Cible injoignable"
		case domain.ErrorStaleTarget:
			prefix = "Cible obsolète"
		case domain.ErrorCancelled:
			prefix = "Opération annulée"
		default:
			prefix = "Podman a refusé l’opération"
		}
		return fmt.Errorf("%s : %s", prefix, podman.ErrorMessage(err))
	}
	return err
}

func isStaleTarget(err error) bool {
	var operation *domain.OperationError
	if errors.As(err, &operation) {
		return operation.Category == domain.ErrorStaleTarget
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "no such container") || strings.Contains(message, "container not found")
}

func actionLabel(action domain.Action) string {
	switch action {
	case domain.ActionStart:
		return "Démarrage"
	case domain.ActionStop:
		return "Arrêt"
	case domain.ActionRestart:
		return "Redémarrage"
	case domain.ActionRemove:
		return "Suppression"
	default:
		return string(action)
	}
}

func shortTarget(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func indexOfProfile(profiles []domain.ConnectionProfile, name string) int {
	for i, profile := range profiles {
		if strings.EqualFold(profile.Name, name) {
			return i
		}
	}
	return 0
}

func clamp(value, low, high int) int {
	if high < low {
		return 0
	}
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}
