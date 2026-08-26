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

	images             []domain.ImageSummary
	imageSelected      int
	imageFilter        string
	imageFiltering     bool
	imageFilterInput   textinput.Model
	imageLoading       bool
	imageDetailLoading bool
	imageDetails       *domain.ImageDetails
	imageDetailTarget  string
	imageFeedback      error

	loading       bool
	detailLoading bool
	connected     bool
	details       *domain.ContainerDetails

	status string
	err    error

	generation    uint64
	requestCancel context.CancelFunc

	pendingAction          domain.Action
	pendingTarget          string
	pendingID              string
	pendingResource        string
	pendingGeneration      uint64
	pendingConnection      string
	pendingContainerCreate domain.ContainerCreateRequest

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

	imagePullInput         textinput.Model
	imagePullReference     string
	imagePullEvents        []domain.ImagePullEvent
	imagePullStatus        domain.ImageOperationStatus
	imagePullError         error
	imagePulling           bool
	imagePullStreamStopped bool
	imagePullGeneration    uint64
	imagePullTarget        string
	imagePullCancel        context.CancelFunc

	containerCreateInputs          []textinput.Model
	containerCreateFocus           int
	containerCreatePrevious        string
	containerCreateRequest         domain.ContainerCreateRequest
	containerCreateImageGeneration uint64
	containerCreateError           error
	containerCreateResult          domain.ContainerRunResult
	containerCreateStatus          domain.ContainerCreateStatus
	containerCreateRunning         bool
	containerCreateRefreshing      bool
	containerCreateGeneration      uint64
	containerCreateTarget          string
}

func New(store *config.Store, factory podman.Factory) Model {
	ctx, cancel := context.WithCancel(context.Background())
	filter := textinput.New()
	filter.Prompt = ""
	filter.Placeholder = "nom, image ou identifiant"
	filter.CharLimit = 120
	filter.SetWidth(36)
	imageFilter := textinput.New()
	imageFilter.Prompt = ""
	imageFilter.Placeholder = "référence, ID ou digest"
	imageFilter.CharLimit = 256
	imageFilter.SetWidth(36)
	pullInput := textinput.New()
	pullInput.Prompt = ""
	pullInput.Placeholder = "registry.example/image:tag"
	pullInput.CharLimit = 256
	pullInput.SetWidth(64)
	createInputs := make([]textinput.Model, 2)
	createPlaceholders := []string{"nom du conteneur", "commande et arguments (optionnel)"}
	for i := range createInputs {
		createInputs[i] = textinput.New()
		createInputs[i].Prompt = ""
		createInputs[i].Placeholder = createPlaceholders[i]
		createInputs[i].CharLimit = 512
		createInputs[i].SetWidth(64)
	}

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
		store:                 store,
		factory:               factory,
		ctx:                   ctx,
		cancel:                cancel,
		file:                  config.Default(),
		keys:                  ui.NewKeyMap(),
		help:                  help.New(),
		screen:                ui.ScreenInventory,
		mode:                  ui.ModeNormal,
		filterInput:           filter,
		imageFilterInput:      imageFilter,
		profileInputs:         inputs,
		viewport:              viewport.New(viewport.WithWidth(80), viewport.WithHeight(12)),
		logFollow:             true,
		imagePullInput:        pullInput,
		imagePullStatus:       domain.ImageOperationIdle,
		containerCreateInputs: createInputs,
		containerCreateStatus: domain.ContainerCreateIdle,
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
		m.stopImagePull()
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
		m.images = nil
		m.imageSelected = 0
		m.imageDetails = nil
		m.imageDetailTarget = ""
		m.imageLoading = false
		m.imageDetailLoading = false
		m.imageFeedback = nil
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
	case ContainerRunFinishedMsg:
		return m.handleContainerRunFinished(message)
	case ContainerCreateRefreshMsg:
		return m.handleContainerCreateRefresh(message)
	case ImageInventoryLoadedMsg:
		if message.Generation != m.generation {
			return m, nil
		}
		m.requestCancel = nil
		m.imageLoading = false
		if message.Err != nil {
			refreshingPull := m.imagePullStatus == domain.ImageOperationRefreshing
			m.err = friendlyError(message.Err)
			if refreshingPull {
				m.imagePullStatus = domain.ImageOperationFailed
				m.imagePullError = friendlyError(message.Err)
				m.status = "Le téléchargement est terminé mais l’inventaire n’a pas pu être actualisé."
			} else {
				m.status = ""
			}
			return m, nil
		}
		m.images = message.Images
		m.imageSelected = clamp(m.imageSelected, 0, len(m.visibleImages())-1)
		feedback := m.imageFeedback
		m.imageFeedback = nil
		m.err = feedback
		if feedback != nil {
			m.status = fmt.Sprintf("%s · inventaire actualisé", feedback)
		} else if m.imagePullStatus == domain.ImageOperationRefreshing {
			m.imagePullStatus = domain.ImageOperationSucceeded
			m.imagePulling = false
			m.screen = ui.ScreenImages
			m.status = fmt.Sprintf("Téléchargement réussi · inventaire actualisé · %d image(s)", len(m.images))
		} else if m.imagePullStatus == domain.ImageOperationCancelled {
			m.status = fmt.Sprintf("Téléchargement annulé · état de l’inventaire vérifié · %d image(s)", len(m.images))
		} else {
			m.status = fmt.Sprintf("Inventaire images actualisé · %d image(s)", len(m.images))
		}
		return m, nil
	case ImageDetailsLoadedMsg:
		if message.Generation != m.generation || message.TargetID != m.imageDetailTarget {
			return m, nil
		}
		m.requestCancel = nil
		m.imageDetailLoading = false
		if message.Err != nil {
			m.err = friendlyError(message.Err)
			return m, nil
		}
		details := mergeImageSummary(message.Details, m.images, message.TargetID)
		m.imageDetails = &details
		m.err = nil
		return m, nil
	case OperationFinishedMsg:
		return m.handleOperation(message)
	case logStreamEvent:
		return m.handleLogEvent(message)
	case statsStreamEvent:
		return m.handleStatsEvent(message)
	case imagePullStreamEvent:
		return m.handleImagePullEvent(message)
	case tea.KeyMsg:
		return m.handleKey(message)
	}
	return m, nil
}

func (m *Model) View() tea.View {
	data := ui.ViewData{
		Width:                         m.width,
		Height:                        m.height,
		Screen:                        m.screen,
		Mode:                          m.mode,
		Profile:                       m.profile,
		Connected:                     m.connected,
		Profiles:                      m.file.Profiles,
		ActiveProfile:                 m.file.Active,
		ProfileCursor:                 m.profileCursor,
		ProfileFields:                 m.profileFieldValues(),
		ProfileFocus:                  m.profileFocus,
		Containers:                    m.visibleContainers(),
		Selected:                      m.selected,
		Filter:                        m.filter,
		FilterEditing:                 m.filtering,
		Loading:                       m.loading || m.detailLoading,
		Images:                        m.visibleImages(),
		ImageSelected:                 m.imageSelected,
		ImageFilter:                   m.imageFilter,
		ImageFilterEditing:            m.imageFiltering,
		ImageLoading:                  m.imageLoading,
		ImageDetailLoading:            m.imageDetailLoading,
		ImageDetails:                  m.imageDetails,
		ImagePullReference:            m.imagePullReference,
		ImagePullInput:                m.imagePullInput.Value(),
		ImagePullInputEditing:         m.screen == ui.ScreenImagePull && !m.imagePulling,
		ImagePullEvents:               append([]domain.ImagePullEvent(nil), m.imagePullEvents...),
		ImagePullStatus:               m.imagePullStatus,
		ImagePullError:                m.imagePullError,
		ImagePulling:                  m.imagePulling,
		ImagePullStopped:              m.imagePullStreamStopped,
		ContainerCreateImageReference: m.containerCreateRequest.ImageReference,
		ContainerCreateImageID:        m.containerCreateRequest.ImageID,
		ContainerCreateFields:         m.containerCreateFieldValues(),
		ContainerCreateFocus:          m.containerCreateFocus,
		ContainerCreateRequest:        m.containerCreateRequest,
		ContainerCreateStatus:         m.containerCreateStatus,
		ContainerCreateError:          m.containerCreateError,
		ContainerCreateResult:         m.containerCreateResult,
		ContainerCreateRunning:        m.containerCreateRunning,
		ContainerCreateRefreshing:     m.containerCreateRefreshing,
		Error:                         m.err,
		Status:                        m.status,
		Details:                       m.details,
		LogContent:                    m.viewport.View(),
		LogFollow:                     m.logFollow,
		StreamStopped:                 m.streamStopped,
		Stats:                         m.stats,
		ConfirmAction:                 actionLabel(m.pendingAction),
		ConfirmTarget:                 m.pendingTarget,
		ConfirmTargetID:               m.pendingID,
		ConfirmResource:               m.pendingResource,
		FormError:                     m.profileFormError,
		Help:                          m.help,
		Keys:                          m.keys,
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
		m.err = friendlyOperationError(message.Action, message.Err)
		m.status = ""
		if isStaleTarget(message.Err) {
			if message.Action == domain.ActionImageRemove {
				m.imageFeedback = m.err
				m.screen = ui.ScreenImages
				m.imageDetails = nil
				m.imageDetailTarget = ""
				return m, m.refreshImages()
			}
			return m, m.refresh()
		}
		return m, nil
	}
	m.err = nil
	m.status = fmt.Sprintf("%s réussi pour %s · actualisation…", actionLabel(message.Action), shortTarget(message.TargetID))
	if message.Action == domain.ActionImageRemove {
		m.screen = ui.ScreenImages
		m.imageDetails = nil
		m.imageDetailTarget = ""
		m.imageFeedback = nil
		return m, m.refreshImages()
	}
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

func (m *Model) handleContainerRunFinished(message ContainerRunFinishedMsg) (tea.Model, tea.Cmd) {
	if message.Generation != m.containerCreateGeneration || message.Target != m.containerCreateTarget {
		return m, nil
	}
	m.requestCancel = nil
	m.containerCreateRunning = false
	m.containerCreateRequest = message.Request
	m.containerCreateResult = message.Result
	m.containerCreateError = nil
	if message.Err != nil {
		m.containerCreateError = friendlyContainerCreateError(message.Err, message.Result)
		if message.Result.ContainerID != "" {
			m.containerCreateStatus = domain.ContainerCreatePartial
			m.status = fmt.Sprintf("Conteneur créé mais non démarré · ID %s · actualisation…", message.Result.ContainerID)
			return m, m.refreshAfterContainerCreate()
		}
		if errors.Is(message.Err, context.Canceled) || errors.Is(message.Err, context.DeadlineExceeded) {
			m.containerCreateStatus = domain.ContainerCreateCancelled
			m.containerCreateError = nil
			m.err = nil
			m.status = "Création annulée ; aucune création confirmée par Podman."
			return m, nil
		}
		m.containerCreateStatus = domain.ContainerCreateFailed
		m.status = ""
		m.err = m.containerCreateError
		return m, nil
	}
	if !message.Result.Started {
		m.containerCreateError = errors.New("Podman n’a pas confirmé le démarrage du conteneur")
		if message.Result.ContainerID != "" {
			m.containerCreateStatus = domain.ContainerCreatePartial
			m.status = fmt.Sprintf("Conteneur créé mais non démarré · ID %s · actualisation…", message.Result.ContainerID)
			return m, m.refreshAfterContainerCreate()
		}
		m.containerCreateStatus = domain.ContainerCreateFailed
		m.status = ""
		m.err = m.containerCreateError
		return m, nil
	}
	m.containerCreateStatus = domain.ContainerCreateRefreshing
	m.status = fmt.Sprintf("Conteneur démarré · actualisation des inventaires pour %s…", shortTarget(message.Result.ContainerID))
	return m, m.refreshAfterContainerCreate()
}

func (m *Model) refreshAfterContainerCreate() tea.Cmd {
	if m.client == nil {
		m.containerCreateRefreshing = false
		m.containerCreateStatus = domain.ContainerCreateFailed
		m.err = errors.New("aucune cible Podman connectée pour actualiser les inventaires")
		return nil
	}
	ctx, generation := m.beginRequest()
	m.containerCreateGeneration = generation
	m.containerCreateRefreshing = true
	m.loading = true
	m.imageLoading = true
	return refreshAfterContainerCreateCmd(ctx, m.client, generation)
}

func (m *Model) handleContainerCreateRefresh(message ContainerCreateRefreshMsg) (tea.Model, tea.Cmd) {
	if !m.containerCreateRefreshing || message.Generation != m.generation || message.Generation != m.containerCreateGeneration {
		return m, nil
	}
	m.requestCancel = nil
	m.containerCreateRefreshing = false
	m.loading = false
	m.imageLoading = false
	m.screen = ui.ScreenInventory
	m.mode = ui.ModeNormal
	m.details = nil

	if message.ContainerErr == nil {
		m.containers = message.Containers
		m.selected = clamp(m.selected, 0, len(m.visibleContainers())-1)
		m.selectContainer(m.containerCreateResult.ContainerID)
	}
	if message.ImageErr == nil {
		m.images = message.Images
		m.imageSelected = clamp(m.imageSelected, 0, len(m.visibleImages())-1)
	}

	var refreshErr error
	if message.ContainerErr != nil {
		refreshErr = fmt.Errorf("conteneurs : %w", friendlyError(message.ContainerErr))
	}
	if message.ImageErr != nil {
		imageErr := fmt.Errorf("images : %w", friendlyError(message.ImageErr))
		if refreshErr == nil {
			refreshErr = imageErr
		} else {
			refreshErr = fmt.Errorf("%v · %v", refreshErr, imageErr)
		}
	}

	if refreshErr != nil {
		m.err = fmt.Errorf("%w · conteneur créé avec l’ID %s", refreshErr, m.containerCreateResult.ContainerID)
		m.status = fmt.Sprintf("Création terminée · actualisation incomplète · ID %s", m.containerCreateResult.ContainerID)
		return m, nil
	}
	m.err = nil
	if m.containerCreateStatus == domain.ContainerCreatePartial {
		m.status = fmt.Sprintf("Conteneur créé mais non démarré · ID %s · inventaires actualisés", m.containerCreateResult.ContainerID)
		if m.containerCreateError != nil {
			m.status += " · " + m.containerCreateError.Error()
		}
	} else {
		m.containerCreateStatus = domain.ContainerCreateSucceeded
		m.status = fmt.Sprintf("Conteneur créé et démarré · ID %s · inventaires actualisés", m.containerCreateResult.ContainerID)
	}
	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.mode == ui.ModeConfirm {
		if key.Matches(msg, m.keys.Confirm) {
			stale := m.pendingConnection != "" && (m.pendingGeneration != m.generation || m.pendingConnection != m.connectionIdentity())
			if m.pendingAction == domain.ActionContainerCreate && (!m.containerCreateImageIsCurrent() || m.containerCreateImageGeneration != m.generation) {
				stale = true
			}
			if stale {
				m.mode = ui.ModeNormal
				wasContainerCreate := m.pendingAction == domain.ActionContainerCreate
				m.resetPendingConfirmation()
				if wasContainerCreate {
					m.containerCreateStatus = domain.ContainerCreateFailed
					m.containerCreateError = errors.New("l’image ou la cible de confirmation n’est plus active")
					m.err = m.containerCreateError
				} else {
					m.err = errors.New("la cible de confirmation n’est plus active")
				}
				return m, nil
			}
			action, id := m.pendingAction, m.pendingID
			request := m.pendingContainerCreate
			m.mode = ui.ModeNormal
			m.resetPendingConfirmation()
			if action == domain.ActionContainerCreate {
				return m, m.runContainerCreate(request)
			}
			return m, m.runOperation(action, id)
		}
		if key.Matches(msg, m.keys.Cancel) {
			wasContainerCreate := m.pendingAction == domain.ActionContainerCreate
			m.mode = ui.ModeNormal
			m.resetPendingConfirmation()
			if wasContainerCreate {
				m.containerCreateStatus = domain.ContainerCreateEditing
				m.containerCreateError = nil
				m.status = "Création annulée ; aucune mutation envoyée."
			} else {
				m.status = "Opération annulée ; la cible n’a pas été modifiée."
			}
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
	if m.imageFiltering {
		if msg.String() == "esc" {
			m.imageFiltering = false
			m.imageFilterInput.Blur()
			return m, nil
		}
		if msg.String() == "enter" {
			m.imageFiltering = false
			m.imageFilterInput.Blur()
			m.imageFilter = m.imageFilterInput.Value()
			m.imageSelected = 0
			return m, nil
		}
		updated, cmd := m.imageFilterInput.Update(msg)
		m.imageFilterInput = updated
		m.imageFilter = updated.Value()
		m.imageSelected = clamp(m.imageSelected, 0, len(m.visibleImages())-1)
		return m, cmd
	}
	if m.screen == ui.ScreenImagePull {
		return m.handleImagePullKey(msg)
	}
	if m.screen == ui.ScreenContainerCreate {
		return m.handleContainerCreateKey(msg)
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
	if key.Matches(msg, m.keys.Images) {
		return m, m.openImages()
	}
	if key.Matches(msg, m.keys.Refresh) {
		if m.screen == ui.ScreenImages {
			return m, m.refreshImages()
		}
		if m.screen == ui.ScreenImageDetails && m.imageDetails != nil && m.client != nil {
			ctx, generation := m.beginRequest()
			m.imageDetailLoading = true
			return m, inspectImageCmd(ctx, m.client, m.imageDetails.ID, generation)
		}
		if m.screen == ui.ScreenDetails && m.details != nil && m.client != nil {
			ctx, generation := m.beginRequest()
			m.detailLoading = true
			return m, inspectContainerCmd(ctx, m.client, m.details.ID, generation)
		}
		return m, m.refresh()
	}
	if key.Matches(msg, m.keys.Filter) {
		if m.screen == ui.ScreenImages {
			m.imageFiltering = true
			m.imageFilterInput.SetValue(m.imageFilter)
			return m, m.imageFilterInput.Focus()
		}
		m.filtering = true
		m.filterInput.SetValue(m.filter)
		return m, m.filterInput.Focus()
	}
	if key.Matches(msg, m.keys.Up) {
		if m.screen == ui.ScreenImages {
			m.imageSelected = clamp(m.imageSelected-1, 0, len(m.visibleImages())-1)
			return m, nil
		}
		if m.screen != ui.ScreenInventory && m.screen != ui.ScreenDetails {
			return m, nil
		}
		m.selected = clamp(m.selected-1, 0, len(m.visibleContainers())-1)
		return m, nil
	}
	if key.Matches(msg, m.keys.Down) {
		if m.screen == ui.ScreenImages {
			m.imageSelected = clamp(m.imageSelected+1, 0, len(m.visibleImages())-1)
			return m, nil
		}
		if m.screen != ui.ScreenInventory && m.screen != ui.ScreenDetails {
			return m, nil
		}
		m.selected = clamp(m.selected+1, 0, len(m.visibleContainers())-1)
		return m, nil
	}
	if key.Matches(msg, m.keys.Open) && m.screen == ui.ScreenInventory {
		return m, m.openDetails()
	}
	if key.Matches(msg, m.keys.Open) && m.screen == ui.ScreenImages {
		return m, m.openImageDetails()
	}
	if key.Matches(msg, m.keys.New) && (m.screen == ui.ScreenImages || m.screen == ui.ScreenImageDetails) {
		return m, m.openContainerCreate()
	}
	if key.Matches(msg, m.keys.Pull) && (m.screen == ui.ScreenImages || m.screen == ui.ScreenImageDetails) {
		return m, m.openImagePull()
	}
	if key.Matches(msg, m.keys.Start) && (m.screen == ui.ScreenInventory || m.screen == ui.ScreenDetails) {
		return m, m.runOperation(domain.ActionStart, m.selectedID())
	}
	if key.Matches(msg, m.keys.Stop) && (m.screen == ui.ScreenInventory || m.screen == ui.ScreenDetails) {
		return m, m.requestConfirmation(domain.ActionStop)
	}
	if key.Matches(msg, m.keys.Restart) && (m.screen == ui.ScreenInventory || m.screen == ui.ScreenDetails) {
		return m, m.requestConfirmation(domain.ActionRestart)
	}
	if key.Matches(msg, m.keys.Remove) {
		if m.screen == ui.ScreenImages || m.screen == ui.ScreenImageDetails {
			return m, m.requestImageConfirmation()
		}
		return m, m.requestConfirmation(domain.ActionRemove)
	}
	if key.Matches(msg, m.keys.Logs) && (m.screen == ui.ScreenInventory || m.screen == ui.ScreenDetails) {
		return m, m.openLogs()
	}
	if key.Matches(msg, m.keys.Stats) && (m.screen == ui.ScreenInventory || m.screen == ui.ScreenDetails) {
		return m, m.openStats()
	}
	if key.Matches(msg, m.keys.Back) && m.screen == ui.ScreenDetails {
		m.screen = ui.ScreenInventory
		m.details = nil
		return m, nil
	}
	if key.Matches(msg, m.keys.Back) && m.screen == ui.ScreenImageDetails {
		m.screen = ui.ScreenImages
		m.imageDetails = nil
		m.imageDetailTarget = ""
		return m, nil
	}
	if key.Matches(msg, m.keys.Back) && m.screen == ui.ScreenImages {
		m.screen = ui.ScreenInventory
		return m, nil
	}
	return m, nil
}

func (m *Model) handleContainerCreateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		m.cancel()
		m.stopStream()
		m.stopImagePull()
		return m, tea.Quit
	}
	if (m.containerCreateRunning || m.containerCreateRefreshing) && key.Matches(msg, m.keys.Quit) {
		m.cancel()
		m.stopStream()
		m.stopImagePull()
		return m, tea.Quit
	}
	if msg.String() == "?" {
		m.help.ShowAll = !m.help.ShowAll
		return m, nil
	}
	if m.containerCreateRunning || m.containerCreateRefreshing {
		if msg.String() == "esc" && m.requestCancel != nil {
			m.requestCancel()
			m.status = "Annulation demandée ; vérification de l’état auprès de Podman…"
		}
		return m, nil
	}
	if msg.String() == "esc" {
		m.blurContainerCreateInputs()
		m.screen = m.containerCreatePrevious
		m.containerCreateStatus = domain.ContainerCreateIdle
		m.containerCreateError = nil
		return m, nil
	}
	if msg.String() == "tab" || msg.String() == "shift+tab" {
		if msg.String() == "shift+tab" {
			m.containerCreateFocus = clamp(m.containerCreateFocus-1, 0, len(m.containerCreateInputs)-1)
		} else {
			m.containerCreateFocus = (m.containerCreateFocus + 1) % len(m.containerCreateInputs)
		}
		return m, m.focusContainerCreateInput()
	}
	if msg.String() == "enter" {
		return m, m.submitContainerCreateForm()
	}
	updated, cmd := m.containerCreateInputs[m.containerCreateFocus].Update(msg)
	m.containerCreateInputs[m.containerCreateFocus] = updated
	m.containerCreateError = updated.Err
	return m, cmd
}

func (m *Model) openContainerCreate() tea.Cmd {
	if m.client == nil {
		m.err = fmt.Errorf("aucune cible Podman connectée")
		return nil
	}
	var image domain.ImageSummary
	var ok bool
	if m.screen == ui.ScreenImageDetails && m.imageDetails != nil {
		image = m.imageDetails.ImageSummary
		if image.ID == "" {
			image.ID = m.imageDetails.ID
		}
		ok = image.ID != ""
	} else {
		image, ok = m.selectedImage()
	}
	if !ok || image.ID == "" {
		m.err = fmt.Errorf("aucune image locale sélectionnée")
		return nil
	}
	m.containerCreatePrevious = m.screen
	m.containerCreateImageGeneration = m.generation
	m.containerCreateRequest = domain.ContainerCreateRequest{
		ImageID:        image.ID,
		ImageReference: image.PrimaryReference(),
	}
	m.containerCreateResult = domain.ContainerRunResult{}
	m.containerCreateError = nil
	m.containerCreateStatus = domain.ContainerCreateEditing
	m.containerCreateRunning = false
	m.containerCreateRefreshing = false
	m.containerCreateFocus = 0
	m.containerCreateInputs[0].SetValue("")
	m.containerCreateInputs[1].SetValue("")
	m.blurContainerCreateInputs()
	m.screen = ui.ScreenContainerCreate
	m.err = nil
	return m.focusContainerCreateInput()
}

func (m *Model) submitContainerCreateForm() tea.Cmd {
	if m.client == nil {
		m.containerCreateError = errors.New("aucune cible Podman connectée")
		return nil
	}
	if m.containerCreateImageGeneration != m.generation || !m.containerCreateImageIsCurrent() {
		m.containerCreateError = errors.New("l’image sélectionnée n’est plus disponible ; actualisez l’inventaire")
		m.containerCreateStatus = domain.ContainerCreateFailed
		return nil
	}
	command, err := domain.ParseContainerCommand(m.containerCreateInputs[1].Value())
	if err != nil {
		m.containerCreateError = err
		m.containerCreateStatus = domain.ContainerCreateEditing
		return nil
	}
	request := domain.ContainerCreateRequest{
		ImageID:        m.containerCreateRequest.ImageID,
		ImageReference: m.containerCreateRequest.ImageReference,
		Name:           m.containerCreateInputs[0].Value(),
		Command:        command,
	}
	if err := request.Validate(); err != nil {
		m.containerCreateError = err
		m.containerCreateStatus = domain.ContainerCreateEditing
		return nil
	}
	m.containerCreateRequest = request
	m.pendingContainerCreate = request
	m.pendingAction = domain.ActionContainerCreate
	m.pendingID = request.ImageID
	m.pendingTarget = request.Name
	m.pendingResource = "container_create"
	m.pendingGeneration = m.containerCreateImageGeneration
	m.pendingConnection = m.connectionIdentity()
	m.containerCreateStatus = domain.ContainerCreateConfirming
	m.containerCreateError = nil
	m.err = nil
	m.mode = ui.ModeConfirm
	return nil
}

func (m *Model) runContainerCreate(request domain.ContainerCreateRequest) tea.Cmd {
	if err := request.Validate(); err != nil {
		m.containerCreateStatus = domain.ContainerCreateFailed
		m.containerCreateError = err
		m.err = err
		return nil
	}
	if m.client == nil {
		m.containerCreateStatus = domain.ContainerCreateFailed
		m.containerCreateError = errors.New("aucune cible Podman connectée")
		m.err = m.containerCreateError
		return nil
	}
	if !m.containerCreateImageIsCurrent() {
		m.containerCreateStatus = domain.ContainerCreateFailed
		m.containerCreateError = errors.New("l’image sélectionnée n’est plus disponible ; création refusée")
		m.err = m.containerCreateError
		return nil
	}
	if m.containerCreateRunning || m.containerCreateRefreshing {
		return nil
	}
	ctx, generation := m.beginRequest()
	m.containerCreateRequest = request
	m.containerCreateResult = domain.ContainerRunResult{}
	m.containerCreateError = nil
	m.containerCreateStatus = domain.ContainerCreateCreating
	m.containerCreateRunning = true
	m.containerCreateTarget = m.connectionIdentity()
	m.containerCreateGeneration = generation
	m.err = nil
	m.status = fmt.Sprintf("Création puis démarrage de %s…", request.Name)
	return runContainerCmd(ctx, m.client, request, m.containerCreateTarget, generation)
}

func (m *Model) focusContainerCreateInput() tea.Cmd {
	m.blurContainerCreateInputs()
	return m.containerCreateInputs[m.containerCreateFocus].Focus()
}

func (m *Model) blurContainerCreateInputs() {
	for i := range m.containerCreateInputs {
		m.containerCreateInputs[i].Blur()
	}
}

func (m *Model) containerCreateFieldValues() []string {
	values := make([]string, len(m.containerCreateInputs))
	for i := range m.containerCreateInputs {
		values[i] = m.containerCreateInputs[i].Value()
	}
	return values
}

func (m *Model) containerCreateImageIsCurrent() bool {
	imageID := m.containerCreateRequest.ImageID
	if imageID == "" {
		return false
	}
	for _, image := range m.images {
		if image.ID == imageID {
			return true
		}
	}
	return false
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
	m.stopImagePull()
	if m.containerCreateRunning || m.containerCreateRefreshing {
		if m.requestCancel != nil {
			m.requestCancel()
		}
	}
	m.containerCreateRunning = false
	m.containerCreateRefreshing = false
	m.containerCreateStatus = domain.ContainerCreateIdle
	m.containerCreateError = nil
	m.containerCreateResult = domain.ContainerRunResult{}
	m.containerCreateRequest = domain.ContainerCreateRequest{}
	m.containerCreateImageGeneration = 0
	m.profile = profile
	m.file.Active = profile.Name
	_ = m.saveConfig()
	m.client = nil
	m.connected = false
	m.screen = ui.ScreenInventory
	m.mode = ui.ModeNormal
	m.details = nil
	m.images = nil
	m.imageSelected = 0
	m.imageDetails = nil
	m.imageDetailTarget = ""
	m.imageLoading = false
	m.imageDetailLoading = false
	m.imagePullStatus = domain.ImageOperationIdle
	m.imagePullError = nil
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

func (m *Model) openImages() tea.Cmd {
	m.screen = ui.ScreenImages
	m.imageDetails = nil
	m.imageDetailTarget = ""
	m.imageFiltering = false
	m.imageFeedback = nil
	m.err = nil
	if m.client == nil {
		m.err = fmt.Errorf("aucune cible Podman connectée")
		return nil
	}
	return m.refreshImages()
}

func (m *Model) refreshImages() tea.Cmd {
	if m.client == nil {
		m.err = fmt.Errorf("aucune cible Podman connectée")
		return nil
	}
	ctx, generation := m.beginRequest()
	m.imageLoading = true
	if m.imagePullStatus != domain.ImageOperationRefreshing && m.imagePullStatus != domain.ImageOperationCancelled {
		m.status = "Actualisation de l’inventaire images…"
	}
	return listImagesCmd(ctx, m.client, generation)
}

func (m *Model) openImageDetails() tea.Cmd {
	image, ok := m.selectedImage()
	if !ok || image.ID == "" || m.client == nil {
		m.err = fmt.Errorf("aucune image sélectionnée")
		return nil
	}
	m.screen = ui.ScreenImageDetails
	m.imageDetails = nil
	m.imageDetailTarget = image.ID
	m.imageDetailLoading = true
	m.err = nil
	ctx, generation := m.beginRequest()
	return inspectImageCmd(ctx, m.client, image.ID, generation)
}

func (m *Model) openImagePull() tea.Cmd {
	if m.client == nil {
		m.err = fmt.Errorf("aucune cible Podman connectée")
		return nil
	}
	m.screen = ui.ScreenImagePull
	m.imagePullInput.SetValue(m.imagePullReference)
	m.imagePullError = nil
	m.err = nil
	return m.imagePullInput.Focus()
}

func (m *Model) handleImagePullKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.Quit) {
		m.cancel()
		m.stopStream()
		m.stopImagePull()
		return m, tea.Quit
	}
	if key.Matches(msg, m.keys.Help) {
		m.help.ShowAll = !m.help.ShowAll
		return m, nil
	}
	if m.imagePulling {
		if msg.String() == "esc" {
			return m, m.cancelImagePull()
		}
		return m, nil
	}
	if msg.String() == "esc" {
		m.imagePullInput.Blur()
		m.screen = ui.ScreenImages
		return m, nil
	}
	if key.Matches(msg, m.keys.Confirm) {
		return m, m.startImagePull()
	}
	updated, cmd := m.imagePullInput.Update(msg)
	m.imagePullInput = updated
	m.imagePullReference = updated.Value()
	m.imagePullError = nil
	return m, cmd
}

func (m *Model) startImagePull() tea.Cmd {
	reference := strings.TrimSpace(m.imagePullInput.Value())
	if reference == "" {
		m.imagePullError = errors.New("la référence d’image ne peut pas être vide")
		return nil
	}
	if m.client == nil {
		m.imagePullError = errors.New("aucune cible Podman connectée")
		return nil
	}
	m.imagePullReference = reference
	m.imagePullInput.SetValue(reference)
	m.imagePullInput.Blur()
	m.stopImagePull()
	m.imagePullGeneration++
	generation := m.imagePullGeneration
	target := m.connectionIdentity()
	ctx, cancel := context.WithCancel(m.ctx)
	m.imagePullCancel = cancel
	m.imagePullTarget = target
	m.imagePullEvents = nil
	m.imagePullStatus = domain.ImageOperationRunning
	m.imagePullError = nil
	m.imagePulling = true
	m.imagePullStreamStopped = false
	m.err = nil
	m.status = fmt.Sprintf("Téléchargement de %s…", reference)
	return startImagePullCmd(ctx, m.client, reference, target, generation)
}

func (m *Model) cancelImagePull() tea.Cmd {
	if !m.imagePulling {
		return nil
	}
	m.stopImagePull()
	m.imagePulling = false
	m.imagePullStreamStopped = true
	m.imagePullStatus = domain.ImageOperationCancelled
	m.imagePullError = nil
	m.status = "Téléchargement annulé · vérification de l’inventaire…"
	return m.refreshImages()
}

func (m *Model) stopImagePull() {
	m.imagePullGeneration++
	if m.imagePullCancel != nil {
		m.imagePullCancel()
		m.imagePullCancel = nil
	}
	m.imagePulling = false
}

func (m *Model) handleImagePullEvent(event imagePullStreamEvent) (tea.Model, tea.Cmd) {
	if event.Generation != m.imagePullGeneration || event.Target != m.imagePullTarget {
		return m, nil
	}
	if event.Event != nil {
		observation := *event.Event
		m.imagePullEvents = append(m.imagePullEvents, observation)
		switch observation.Kind {
		case domain.ImagePullError:
			m.imagePullStatus = domain.ImageOperationFailed
			if strings.TrimSpace(observation.Text) != "" {
				m.imagePullError = errors.New(observation.Text)
			}
		case domain.ImagePullCancelled:
			m.imagePullStatus = domain.ImageOperationCancelled
		}
	}
	if event.Done {
		m.imagePulling = false
		m.imagePullStreamStopped = true
		m.imagePullCancel = nil
		if event.Err != nil {
			if errors.Is(event.Err, context.Canceled) || errors.Is(event.Err, context.DeadlineExceeded) {
				m.imagePullStatus = domain.ImageOperationCancelled
				m.imagePullError = nil
				m.status = "Téléchargement annulé · l’inventaire reste à vérifier."
				return m, nil
			}
			m.imagePullStatus = domain.ImageOperationFailed
			m.imagePullError = friendlyImagePullError(event.Err)
			m.status = "Le téléchargement a échoué ; la progression reçue est conservée."
			return m, nil
		}
		if m.imagePullStatus == domain.ImageOperationFailed || m.imagePullStatus == domain.ImageOperationCancelled {
			return m, nil
		}
		m.imagePullStatus = domain.ImageOperationRefreshing
		m.status = "Téléchargement terminé · actualisation de l’inventaire…"
		return m, m.refreshImages()
	}
	if event.Next != nil {
		return m, waitImagePullStream(event.Next, event.Target, event.Generation)
	}
	return m, nil
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
		m.err = fmt.Errorf("aucune cible sélectionnée")
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
	m.pendingResource = "conteneur"
	m.pendingGeneration = m.generation
	m.pendingConnection = m.connectionIdentity()
	m.mode = ui.ModeConfirm
	m.err = nil
	return nil
}

func (m *Model) requestImageConfirmation() tea.Cmd {
	image, ok := m.selectedImage()
	if m.screen == ui.ScreenImageDetails && m.imageDetails != nil {
		image = m.imageDetails.ImageSummary
		ok = image.ID != ""
	}
	if !ok || image.ID == "" {
		m.err = fmt.Errorf("aucune image sélectionnée")
		return nil
	}
	m.pendingAction = domain.ActionImageRemove
	m.pendingID = image.ID
	m.pendingTarget = imageTargetName(image)
	m.pendingResource = "image"
	m.pendingGeneration = m.generation
	m.pendingConnection = m.connectionIdentity()
	m.mode = ui.ModeConfirm
	m.err = nil
	return nil
}

func (m *Model) resetPendingConfirmation() {
	m.pendingAction = ""
	m.pendingTarget = ""
	m.pendingID = ""
	m.pendingResource = ""
	m.pendingGeneration = 0
	m.pendingConnection = ""
	m.pendingContainerCreate = domain.ContainerCreateRequest{}
}

func (m *Model) connectionIdentity() string {
	return strings.Join([]string{m.profile.Name, m.profile.URI, m.profile.IdentityPath}, "\x00")
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

func (m *Model) visibleImages() []domain.ImageSummary {
	query := strings.ToLower(strings.TrimSpace(m.imageFilter))
	if query == "" {
		return append([]domain.ImageSummary(nil), m.images...)
	}
	result := make([]domain.ImageSummary, 0, len(m.images))
	for _, image := range m.images {
		fields := append([]string{image.ID, image.Digest}, image.References...)
		fields = append(fields, image.Digests...)
		for _, field := range fields {
			if strings.Contains(strings.ToLower(field), query) {
				result = append(result, image)
				break
			}
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

func (m *Model) selectContainer(id string) {
	if id == "" {
		return
	}
	for i, container := range m.visibleContainers() {
		if container.ID == id {
			m.selected = i
			return
		}
	}
}

func (m *Model) selectedImage() (domain.ImageSummary, bool) {
	images := m.visibleImages()
	if m.imageSelected < 0 || m.imageSelected >= len(images) {
		return domain.ImageSummary{}, false
	}
	return images[m.imageSelected], true
}

func (m *Model) selectedID() string {
	if m.screen == ui.ScreenImageDetails && m.imageDetails != nil {
		return m.imageDetails.ID
	}
	if m.screen == ui.ScreenImages || m.screen == ui.ScreenImageDetails {
		image, ok := m.selectedImage()
		if !ok {
			return ""
		}
		return image.ID
	}
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
	if m.screen == ui.ScreenImageDetails && m.imageDetails != nil {
		return imageTargetName(m.imageDetails.ImageSummary)
	}
	if m.screen == ui.ScreenImages || m.screen == ui.ScreenImageDetails {
		image, ok := m.selectedImage()
		if !ok {
			return "image inconnue"
		}
		return imageTargetName(image)
	}
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

func imageTargetName(image domain.ImageSummary) string {
	if reference := image.PrimaryReference(); reference != "" {
		return reference
	}
	if image.ID != "" {
		return shortTarget(image.ID)
	}
	return "image inconnue"
}

func mergeImageSummary(details domain.ImageDetails, images []domain.ImageSummary, target string) domain.ImageDetails {
	for _, image := range images {
		if image.ID == target {
			details.Containers = image.Containers
			break
		}
	}
	return details
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
		case domain.ErrorRegistry:
			prefix = "Registre indisponible ou référence refusée"
		case domain.ErrorInUse:
			prefix = "Image utilisée par un conteneur"
		case domain.ErrorNameConflict:
			prefix = "Nom de conteneur déjà utilisé"
		case domain.ErrorInvalidConfig:
			prefix = "Configuration invalide"
		case domain.ErrorPartial:
			prefix = "Création partielle"
		case domain.ErrorMalformedStream:
			prefix = "Flux Podman invalide"
		case domain.ErrorCancelled:
			prefix = "Opération annulée"
		default:
			prefix = "Podman a refusé l’opération"
		}
		return fmt.Errorf("%s : %s", prefix, podman.ErrorMessage(err))
	}
	return err
}

func friendlyContainerCreateError(err error, result domain.ContainerRunResult) error {
	if err == nil {
		return nil
	}
	var operation *domain.OperationError
	if errors.As(err, &operation) && operation.Category == domain.ErrorPartial {
		message := operation.Err
		if message == nil {
			message = errors.New("le démarrage n’a pas abouti")
		}
		return fmt.Errorf("créé mais non démarré · ID %s : %s", result.ContainerID, friendlyError(message))
	}
	return friendlyOperationError(domain.ActionContainerCreate, err)
}

func friendlyImagePullError(err error) error {
	if err == nil {
		return nil
	}
	var operation *domain.OperationError
	if errors.As(err, &operation) {
		return friendlyError(err)
	}
	message := strings.ToLower(err.Error())
	var prefix string
	switch {
	case strings.Contains(message, "manifest unknown"), strings.Contains(message, "name unknown"), strings.Contains(message, "repository does not exist"), strings.Contains(message, "rate limit"):
		prefix = "Registre indisponible ou référence refusée"
	case strings.Contains(message, "unauthorized"), strings.Contains(message, "permission denied"), strings.Contains(message, "authentication"):
		prefix = "Autorisation refusée"
	case strings.Contains(message, "failed to decode"), strings.Contains(message, "unexpected input"):
		prefix = "Flux Podman invalide"
	default:
		return err
	}
	return fmt.Errorf("%s : %s", prefix, err)
}

func friendlyOperationError(action domain.Action, err error) error {
	if action == domain.ActionImageRemove {
		if err == nil {
			return nil
		}
		var operation *domain.OperationError
		if errors.As(err, &operation) {
			return friendlyError(err)
		}
		message := strings.ToLower(err.Error())
		switch {
		case strings.Contains(message, "in use"), strings.Contains(message, "being used"), strings.Contains(message, "used by"):
			return fmt.Errorf("Image utilisée par un conteneur : %s", err)
		case strings.Contains(message, "no such image"), strings.Contains(message, "image not found"):
			return fmt.Errorf("Cible obsolète : %s", err)
		case strings.Contains(message, "permission denied"), strings.Contains(message, "unauthorized"), strings.Contains(message, "forbidden"), strings.Contains(message, "authentication"):
			return fmt.Errorf("Autorisation refusée : %s", err)
		}
	}
	return friendlyError(err)
}

func isStaleTarget(err error) bool {
	var operation *domain.OperationError
	if errors.As(err, &operation) {
		return operation.Category == domain.ErrorStaleTarget
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "no such container") || strings.Contains(message, "container not found") || strings.Contains(message, "no such image") || strings.Contains(message, "image not found")
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
	case domain.ActionImageRemove:
		return "Suppression"
	case domain.ActionContainerCreate:
		return "Création de conteneur"
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
