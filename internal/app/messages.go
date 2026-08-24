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
			default:
				err = context.Canceled
			}
		}
		return OperationFinishedMsg{Generation: generation, Action: action, TargetID: id, Err: err}
	}
}

func startLogStreamCmd(ctx context.Context, client podman.Client, id string, options podman.LogOptions, generation uint64) tea.Cmd {
	return func() tea.Msg {
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
	return func() tea.Msg {
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
