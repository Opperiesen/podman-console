package fixtures

import (
	"context"
	"errors"
	"fmt"
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

	Containers   []domain.ContainerSummary
	Details      map[string]domain.ContainerDetails
	Images       []domain.ImageSummary
	ImageDetails map[string]domain.ImageDetails
	PullEvents   []domain.ImagePullEvent
	PullErr      error
	PullWait     <-chan struct{}
	PullFunc     func(context.Context, string, func(domain.ImagePullEvent)) error
	Logs         []domain.LogLine
	Stats        []domain.ContainerStats
	Errors       map[domain.Action]error
	StreamErr    error
	RunResult    domain.ContainerRunResult
	RunFunc      func(context.Context, domain.ContainerCreateRequest) (domain.ContainerRunResult, error)

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

func (c *Client) RunContainer(ctx context.Context, request domain.ContainerCreateRequest) (domain.ContainerRunResult, error) {
	if err := ctx.Err(); err != nil {
		return domain.ContainerRunResult{}, err
	}
	if err := request.Validate(); err != nil {
		return domain.ContainerRunResult{}, err
	}
	c.mu.Lock()
	c.Calls = append(c.Calls, fmt.Sprintf("%s:%s:%s", domain.ActionContainerCreate, request.ImageID, request.Name))
	runFunc := c.RunFunc
	createErr := c.Errors[domain.ActionContainerCreate]
	startErr := c.Errors[domain.ActionStart]
	result := c.RunResult
	c.mu.Unlock()
	if runFunc != nil {
		return runFunc(ctx, request)
	}
	if createErr != nil {
		return domain.ContainerRunResult{}, createErr
	}
	if err := ctx.Err(); err != nil {
		return domain.ContainerRunResult{}, err
	}
	if result.ContainerID == "" {
		result.ContainerID = "created-container"
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Calls = append(c.Calls, string(domain.ActionStart)+":"+result.ContainerID)
	image := request.ImageReference
	if image == "" {
		image = request.ImageID
	}
	container := domain.ContainerSummary{ID: result.ContainerID, Name: request.Name, Image: image, State: domain.StateCreated}
	found := false
	for _, existing := range c.Containers {
		if existing.ID == container.ID {
			found = true
			break
		}
	}
	if !found {
		c.Containers = append(c.Containers, container)
	}
	if c.Details == nil {
		c.Details = map[string]domain.ContainerDetails{}
	}
	if _, ok := c.Details[container.ID]; !ok {
		c.Details[container.ID] = domain.ContainerDetails{ContainerSummary: container, Command: append([]string(nil), request.Command...)}
	}
	if startErr != nil {
		result.Started = false
		return result, startErr
	}
	result.Started = true
	for i := range c.Containers {
		if c.Containers[i].ID == container.ID {
			c.Containers[i].State = domain.StateRunning
			break
		}
	}
	return result, nil
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

func (c *Client) ListImages(ctx context.Context) ([]domain.ImageSummary, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Calls = append(c.Calls, string(domain.ActionImageList))
	if err := c.Errors[domain.ActionImageList]; err != nil {
		return nil, err
	}
	return append([]domain.ImageSummary(nil), c.Images...), nil
}

func (c *Client) InspectImage(ctx context.Context, id string) (domain.ImageDetails, error) {
	if err := ctx.Err(); err != nil {
		return domain.ImageDetails{}, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Calls = append(c.Calls, string(domain.ActionImageInspect)+":"+id)
	if err := c.Errors[domain.ActionImageInspect]; err != nil {
		return domain.ImageDetails{}, err
	}
	details, ok := c.ImageDetails[id]
	if !ok {
		return domain.ImageDetails{}, errors.New("no such image")
	}
	return details, nil
}

func (c *Client) PullImage(ctx context.Context, reference string, emit func(domain.ImagePullEvent)) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	c.Calls = append(c.Calls, string(domain.ActionImagePull)+":"+reference)
	pullEvents := append([]domain.ImagePullEvent(nil), c.PullEvents...)
	pullErr := c.PullErr
	pullWait := c.PullWait
	pullFunc := c.PullFunc
	configuredErr := c.Errors[domain.ActionImagePull]
	c.mu.Unlock()
	if pullFunc != nil {
		return pullFunc(ctx, reference, emit)
	}
	if pullWait != nil {
		select {
		case <-pullWait:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	for _, event := range pullEvents {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if event.Reference == "" {
			event.Reference = reference
		}
		if emit != nil {
			emit(event)
		}
	}
	if configuredErr != nil {
		return configuredErr
	}
	return pullErr
}

func (c *Client) RemoveImage(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Calls = append(c.Calls, string(domain.ActionImageRemove)+":"+id)
	if err := c.Errors[domain.ActionImageRemove]; err != nil {
		return err
	}
	index := -1
	for i := range c.Images {
		if c.Images[i].ID == id {
			index = i
			break
		}
	}
	if index < 0 {
		return errors.New("no such image")
	}
	c.Images = append(c.Images[:index], c.Images[index+1:]...)
	delete(c.ImageDetails, id)
	return nil
}
