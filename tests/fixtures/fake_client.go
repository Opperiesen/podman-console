package fixtures

import (
	"context"
	"errors"
	"sync"

	"github.com/Opperiesen/podman-console/internal/domain"
	"github.com/Opperiesen/podman-console/internal/podman"
)

// Factory and Client are deterministic test doubles for the application and
// acceptance tests. They never contact a Podman host.
type Factory struct {
	Clients map[string]*Client
	Errors  map[string]error
}

func (f *Factory) Connect(_ context.Context, profile domain.ConnectionProfile) (podman.Client, error) {
	if err := f.Errors[profile.Name]; err != nil {
		return nil, err
	}
	client := f.Clients[profile.Name]
	if client == nil {
		return nil, errors.New("fake profile not found")
	}
	return client, nil
}

type Client struct {
	mu sync.Mutex

	Containers []domain.ContainerSummary
	Details    map[string]domain.ContainerDetails
	Logs       []domain.LogLine
	Stats      []domain.ContainerStats
	Errors     map[domain.Action]error
	StreamErr  error

	Calls []string
}

func (c *Client) ListContainers(ctx context.Context) ([]domain.ContainerSummary, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Calls = append(c.Calls, string(domain.ActionList))
	return append([]domain.ContainerSummary(nil), c.Containers...), nil
}

func (c *Client) InspectContainer(ctx context.Context, id string) (domain.ContainerDetails, error) {
	if err := ctx.Err(); err != nil {
		return domain.ContainerDetails{}, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Calls = append(c.Calls, string(domain.ActionInspect)+":"+id)
	if err := c.Errors[domain.ActionInspect]; err != nil {
		return domain.ContainerDetails{}, err
	}
	details, ok := c.Details[id]
	if !ok {
		return domain.ContainerDetails{}, errors.New("no such container")
	}
	return details, nil
}

func (c *Client) lifecycle(ctx context.Context, action domain.Action, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Calls = append(c.Calls, string(action)+":"+id)
	if err := c.Errors[action]; err != nil {
		return err
	}
	for i := range c.Containers {
		if c.Containers[i].ID != id {
			continue
		}
		switch action {
		case domain.ActionStart:
			c.Containers[i].State = domain.StateRunning
		case domain.ActionStop:
			c.Containers[i].State = domain.StateStopped
		case domain.ActionRestart:
			c.Containers[i].State = domain.StateRunning
		case domain.ActionRemove:
			c.Containers = append(c.Containers[:i], c.Containers[i+1:]...)
		}
		return nil
	}
	return errors.New("no such container")
}

func (c *Client) Start(ctx context.Context, id string) error {
	return c.lifecycle(ctx, domain.ActionStart, id)
}

func (c *Client) Stop(ctx context.Context, id string) error {
	return c.lifecycle(ctx, domain.ActionStop, id)
}

func (c *Client) Restart(ctx context.Context, id string) error {
	return c.lifecycle(ctx, domain.ActionRestart, id)
}

func (c *Client) Remove(ctx context.Context, id string) error {
	return c.lifecycle(ctx, domain.ActionRemove, id)
}

func (c *Client) StreamLogs(ctx context.Context, id string, _ podman.LogOptions, emit func(domain.LogLine)) error {
	c.mu.Lock()
	c.Calls = append(c.Calls, string(domain.ActionLogs)+":"+id)
	lines := append([]domain.LogLine(nil), c.Logs...)
	err := c.StreamErr
	c.mu.Unlock()
	for _, line := range lines {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			emit(line)
		}
	}
	if err != nil {
		return err
	}
	<-ctx.Done()
	return ctx.Err()
}

func (c *Client) StreamStats(ctx context.Context, id string, emit func(domain.ContainerStats)) error {
	c.mu.Lock()
	c.Calls = append(c.Calls, string(domain.ActionStats)+":"+id)
	samples := append([]domain.ContainerStats(nil), c.Stats...)
	err := c.StreamErr
	c.mu.Unlock()
	for _, sample := range samples {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			emit(sample)
		}
	}
	if err != nil {
		return err
	}
	<-ctx.Done()
	return ctx.Err()
}
