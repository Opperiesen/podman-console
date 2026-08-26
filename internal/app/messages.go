package app

import (
	"context"

	"charm.land/bubbletea/v2"
	"github.com/Opperiesen/podman-console/internal/config"
	"github.com/Opperiesen/podman-console/internal/domain"
	"github.com/Opperiesen/podman-console/internal/podman"
)

type ConfigLoadedMsg struct {
	File config.File
	Err  error
}

type ProfileConnectedMsg struct {
	Generation uint64
	Profile    domain.ConnectionProfile
	Client     podman.Client
	Containers []domain.ContainerSummary
	Err        error
}

type InventoryLoadedMsg struct {
	Generation uint64
	Containers []domain.ContainerSummary
	Err        error
}

type DetailsLoadedMsg struct {
	Generation uint64
	Details    domain.ContainerDetails
	Err        error
}

type ImageInventoryLoadedMsg struct {
	Generation uint64
	Images     []domain.ImageSummary
	Err        error
}

type ImageDetailsLoadedMsg struct {
	Generation uint64
	TargetID   string
	Details    domain.ImageDetails
	Err        error
}

type OperationFinishedMsg struct {
	Generation uint64
	Action     domain.Action
	TargetID   string
	Err        error
}

type logStreamEvent struct {
	Generation uint64
	Line       *domain.LogLine
	Err        error
	Done       bool
	Next       <-chan logStreamEvent
}

type statsStreamEvent struct {
	Generation uint64
	Sample     *domain.ContainerStats
	Err        error
	Done       bool
	Next       <-chan statsStreamEvent
}

type imagePullStreamEvent struct {
	Generation uint64
	Target     string
	Event      *domain.ImagePullEvent
	Err        error
	Done       bool
	Next       <-chan imagePullStreamEvent
}

func loadConfigCmd(store *config.Store) tea.Cmd {
	return func() tea.Msg {
		if store == nil {
			return ConfigLoadedMsg{File: config.Default()}
		}
		file, err := store.Load()
		return ConfigLoadedMsg{File: file, Err: err}
	}
}

func connectProfileCmd(ctx context.Context, factory podman.Factory, profile domain.ConnectionProfile, generation uint64) tea.Cmd {
	return func() tea.Msg {
		if factory == nil {
			return ProfileConnectedMsg{Generation: generation, Profile: profile, Err: context.Canceled}
		}
		client, err := factory.Connect(ctx, profile)
		if err != nil {
			return ProfileConnectedMsg{Generation: generation, Profile: profile, Err: err}
		}
		containers, err := client.ListContainers(ctx)
		return ProfileConnectedMsg{Generation: generation, Profile: profile, Client: client, Containers: containers, Err: err}
	}
}

func listContainersCmd(ctx context.Context, client podman.Client, generation uint64) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return InventoryLoadedMsg{Generation: generation, Err: context.Canceled}
		}
		containers, err := client.ListContainers(ctx)
		return InventoryLoadedMsg{Generation: generation, Containers: containers, Err: err}
	}
}

func inspectContainerCmd(ctx context.Context, client podman.Client, id string, generation uint64) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return DetailsLoadedMsg{Generation: generation, Err: context.Canceled}
		}
		details, err := client.InspectContainer(ctx, id)
		return DetailsLoadedMsg{Generation: generation, Details: details, Err: err}
	}
}

func listImagesCmd(ctx context.Context, client podman.Client, generation uint64) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return ImageInventoryLoadedMsg{Generation: generation, Err: context.Canceled}
		}
		images, err := client.ListImages(ctx)
		return ImageInventoryLoadedMsg{Generation: generation, Images: images, Err: err}
	}
}

func inspectImageCmd(ctx context.Context, client podman.Client, id string, generation uint64) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return ImageDetailsLoadedMsg{Generation: generation, TargetID: id, Err: context.Canceled}
		}
		details, err := client.InspectImage(ctx, id)
		return ImageDetailsLoadedMsg{Generation: generation, TargetID: id, Details: details, Err: err}
	}
}

func operationCmd(ctx context.Context, client podman.Client, action domain.Action, id string, generation uint64) tea.Cmd {
	return func() tea.Msg {
		var err error
		if client == nil {
			err = context.Canceled
		} else {
			switch action {
			case domain.ActionStart:
				err = client.Start(ctx, id)
			case domain.ActionStop:
				err = client.Stop(ctx, id)
			case domain.ActionRestart:
				err = client.Restart(ctx, id)
			case domain.ActionRemove:
				err = client.Remove(ctx, id)
			case domain.ActionImageRemove:
				err = client.RemoveImage(ctx, id)
			default:
				err = context.Canceled
			}
		}
		return OperationFinishedMsg{Generation: generation, Action: action, TargetID: id, Err: err}
	}
}

func startImagePullCmd(ctx context.Context, client podman.Client, reference, target string, generation uint64) tea.Cmd {
	channel := make(chan imagePullStreamEvent, 1)
	go func() {
		var err error
		if client == nil {
			err = context.Canceled
		} else {
			err = client.PullImage(ctx, reference, func(event domain.ImagePullEvent) {
				event.Target = target
				if event.Reference == "" {
					event.Reference = reference
				}
				select {
				case channel <- imagePullStreamEvent{Generation: generation, Target: target, Event: &event}:
				case <-ctx.Done():
				}
			})
		}
		select {
		case channel <- imagePullStreamEvent{Generation: generation, Target: target, Done: true, Err: err}:
		case <-ctx.Done():
			select {
			case channel <- imagePullStreamEvent{Generation: generation, Target: target, Done: true, Err: ctx.Err()}:
			default:
			}
		}
		close(channel)
	}()
	return waitImagePullStream(channel, target, generation)
}

func waitImagePullStream(channel <-chan imagePullStreamEvent, target string, generation uint64) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-channel
		if !ok {
			return imagePullStreamEvent{Generation: generation, Target: target, Done: true}
		}
		event.Generation = generation
		event.Target = target
		event.Next = channel
		return event
	}
}

func startLogStreamCmd(ctx context.Context, client podman.Client, id string, options podman.LogOptions, generation uint64) tea.Cmd {
	channel := make(chan logStreamEvent, 1)
	go func() {
		err := client.StreamLogs(ctx, id, options, func(line domain.LogLine) {
			select {
			case channel <- logStreamEvent{Generation: generation, Line: &line}:
			case <-ctx.Done():
			}
		})
		select {
		case channel <- logStreamEvent{Generation: generation, Done: true, Err: err}:
		case <-ctx.Done():
			select {
			case channel <- logStreamEvent{Generation: generation, Done: true, Err: ctx.Err()}:
			default:
			}
		}
		close(channel)
	}()
	return waitLogStream(channel, generation)
}

func waitLogStream(channel <-chan logStreamEvent, generation uint64) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-channel
		if !ok {
			return logStreamEvent{Generation: generation, Done: true}
		}
		event.Generation = generation
		event.Next = channel
		return event
	}
}

func startStatsStreamCmd(ctx context.Context, client podman.Client, id string, generation uint64) tea.Cmd {
	channel := make(chan statsStreamEvent, 1)
	go func() {
		err := client.StreamStats(ctx, id, func(sample domain.ContainerStats) {
			select {
			case channel <- statsStreamEvent{Generation: generation, Sample: &sample}:
			case <-ctx.Done():
			}
		})
		select {
		case channel <- statsStreamEvent{Generation: generation, Done: true, Err: err}:
		case <-ctx.Done():
			select {
			case channel <- statsStreamEvent{Generation: generation, Done: true, Err: ctx.Err()}:
			default:
			}
		}
		close(channel)
	}()
	return waitStatsStream(channel, generation)
}

func waitStatsStream(channel <-chan statsStreamEvent, generation uint64) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-channel
		if !ok {
			return statsStreamEvent{Generation: generation, Done: true}
		}
		event.Generation = generation
		event.Next = channel
		return event
	}
}
