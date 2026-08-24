package podman

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Opperiesen/podman-console/internal/domain"
	networktypes "go.podman.io/common/libnetwork/types"
	"go.podman.io/podman/v6/libpod/define"
	"go.podman.io/podman/v6/pkg/bindings"
	"go.podman.io/podman/v6/pkg/bindings/containers"
)

type BindingsFactory struct{}

func (BindingsFactory) Connect(ctx context.Context, profile domain.ConnectionProfile) (Client, error) {
	if err := profile.Validate(); err != nil {
		return nil, &domain.OperationError{Category: domain.ErrorInvalidConfig, Action: domain.ActionList, Err: err}
	}
	connectionCtx, err := bindings.NewConnectionWithIdentity(ctx, profile.URI, profile.IdentityPath, false)
	if err != nil {
		return nil, Wrap(domain.ActionList, profile.Name, err)
	}
	return &bindingsClient{base: connectionCtx}, nil
}

type bindingsClient struct {
	base context.Context
}

func (c *bindingsClient) operationContext(ctx context.Context) (context.Context, func()) {
	opCtx, cancel := context.WithCancel(context.WithoutCancel(c.base))
	stop := context.AfterFunc(ctx, cancel)
	return opCtx, func() {
		stop()
		cancel()
	}
}

func (c *bindingsClient) ListContainers(ctx context.Context) ([]domain.ContainerSummary, error) {
	opCtx, done := c.operationContext(ctx)
	defer done()
	all := true
	rows, err := containers.List(opCtx, &containers.ListOptions{All: &all})
	if err != nil {
		return nil, Wrap(domain.ActionList, "", err)
	}
	result := make([]domain.ContainerSummary, 0, len(rows))
	for _, row := range rows {
		name := ""
		if len(row.Names) > 0 {
			name = strings.TrimPrefix(row.Names[0], "/")
		}
		result = append(result, domain.ContainerSummary{
			ID:        row.ID,
			Name:      name,
			Image:     row.Image,
			State:     domain.NormalizeContainerState(row.State),
			Status:    row.Status,
			Ports:     convertPorts(row.Ports),
			CreatedAt: row.Created,
		})
	}
	return result, nil
}

func (c *bindingsClient) InspectContainer(ctx context.Context, id string) (domain.ContainerDetails, error) {
	opCtx, done := c.operationContext(ctx)
	defer done()
	inspect, err := containers.Inspect(opCtx, id, nil)
	if err != nil {
		return domain.ContainerDetails{}, Wrap(domain.ActionInspect, id, err)
	}
	if inspect == nil {
		return domain.ContainerDetails{}, Wrap(domain.ActionInspect, id, fmt.Errorf("empty inspect response"))
	}
	state := domain.StateUnknown
	status := ""
	if inspect.State != nil {
		state = domain.NormalizeContainerState(inspect.State.Status)
		status = inspect.State.Status
	}
	name := strings.TrimPrefix(inspect.Name, "/")
	details := domain.ContainerDetails{
		ContainerSummary: domain.ContainerSummary{
			ID:        inspect.ID,
			Name:      name,
			Image:     firstNonEmpty(inspect.ImageName, inspect.Image),
			State:     state,
			Status:    status,
			CreatedAt: inspect.Created,
		},
		Labels: map[string]string{},
	}
	if inspect.Config != nil {
		details.Command = append([]string(nil), inspect.Config.Cmd...)
		details.Entrypoint = append([]string(nil), inspect.Config.Entrypoint...)
		details.WorkingDir = inspect.Config.WorkingDir
		for key, value := range inspect.Config.Labels {
			details.Labels[key] = value
		}
	}
	for _, mount := range inspect.Mounts {
		details.Mounts = append(details.Mounts, domain.Mount{
			Type: mount.Type, Source: mount.Source, Destination: mount.Destination,
			Mode: mount.Mode, ReadWrite: mount.RW,
		})
	}
	if inspect.NetworkSettings != nil {
		networkNames := make([]string, 0, len(inspect.NetworkSettings.Networks))
		for name := range inspect.NetworkSettings.Networks {
			networkNames = append(networkNames, name)
		}
		sort.Strings(networkNames)
		for _, name := range networkNames {
			network := inspect.NetworkSettings.Networks[name]
			attachment := domain.NetworkAttachment{Name: name}
			if network != nil {
				attachment.IPAddress = network.IPAddress
				attachment.MACAddress = network.MacAddress
			}
			details.Networks = append(details.Networks, attachment)
		}
		details.ContainerSummary.Ports = convertInspectPorts(inspect.NetworkSettings.Ports)
	}
	return details, nil
}

func (c *bindingsClient) Start(ctx context.Context, id string) error {
	return c.lifecycle(ctx, domain.ActionStart, id, func(opCtx context.Context) error {
		return containers.Start(opCtx, id, nil)
	})
}

func (c *bindingsClient) Stop(ctx context.Context, id string) error {
	return c.lifecycle(ctx, domain.ActionStop, id, func(opCtx context.Context) error {
		return containers.Stop(opCtx, id, nil)
	})
}

func (c *bindingsClient) Restart(ctx context.Context, id string) error {
	return c.lifecycle(ctx, domain.ActionRestart, id, func(opCtx context.Context) error {
		return containers.Restart(opCtx, id, nil)
	})
}

func (c *bindingsClient) Remove(ctx context.Context, id string) error {
	return c.lifecycle(ctx, domain.ActionRemove, id, func(opCtx context.Context) error {
		_, err := containers.Remove(opCtx, id, nil)
		return err
	})
}

func (c *bindingsClient) lifecycle(ctx context.Context, action domain.Action, id string, fn func(context.Context) error) error {
	opCtx, done := c.operationContext(ctx)
	defer done()
	return Wrap(action, id, fn(opCtx))
}

func (c *bindingsClient) StreamLogs(ctx context.Context, id string, options LogOptions, emit func(domain.LogLine)) error {
	opCtx, finish := c.operationContext(ctx)
	defer finish()
	follow, tail, timestamps := options.Follow, strconv.Itoa(options.Tail), options.Timestamps
	stdout := true
	stderr := true
	logOptions := &containers.LogOptions{Follow: &follow, Tail: &tail, Timestamps: &timestamps, Stdout: &stdout, Stderr: &stderr}
	stdoutCh := make(chan string)
	stderrCh := make(chan string)
	errCh := make(chan error, 1)
	go func() { errCh <- containers.Logs(opCtx, id, logOptions, stdoutCh, stderrCh) }()
	streamDone := false
	for stdoutCh != nil || stderrCh != nil || !streamDone {
		select {
		case line, ok := <-stdoutCh:
			if !ok {
				stdoutCh = nil
				continue
			}
			emit(domain.LogLine{Text: line, Stream: "stdout", ObservedAt: time.Now()})
		case line, ok := <-stderrCh:
			if !ok {
				stderrCh = nil
				continue
			}
			emit(domain.LogLine{Text: line, Stream: "stderr", ObservedAt: time.Now()})
		case err := <-errCh:
			streamDone = true
			if err != nil {
				return Wrap(domain.ActionLogs, id, err)
			}
		case <-ctx.Done():
			return Wrap(domain.ActionLogs, id, ctx.Err())
		}
	}
	return nil
}

func (c *bindingsClient) StreamStats(ctx context.Context, id string, emit func(domain.ContainerStats)) error {
	opCtx, done := c.operationContext(ctx)
	defer done()
	stream := true
	reports, err := containers.Stats(opCtx, []string{id}, &containers.StatsOptions{Stream: &stream})
	if err != nil {
		return Wrap(domain.ActionStats, id, err)
	}
	for {
		select {
		case report, ok := <-reports:
			if !ok {
				return nil
			}
			if report.Error != nil {
				return Wrap(domain.ActionStats, id, report.Error)
			}
			for _, stat := range report.Stats {
				emit(domain.ContainerStats{
					ContainerID: stat.ContainerID, CPUPercent: stat.CPU, MemoryUsageBytes: stat.MemUsage,
					MemoryLimitBytes: stat.MemLimit, MemoryPercent: stat.MemPerc, ObservedAt: time.Now(),
				})
			}
		case <-ctx.Done():
			return Wrap(domain.ActionStats, id, ctx.Err())
		}
	}
}

func convertPorts(ports []networktypes.PortMapping) []domain.PortMapping {
	result := make([]domain.PortMapping, 0, len(ports))
	for _, port := range ports {
		result = append(result, domain.PortMapping{
			HostIP: port.HostIP, HostPort: port.HostPort, ContainerPort: port.ContainerPort, Protocol: port.Protocol,
		})
	}
	return result
}

func convertInspectPorts(ports map[string][]define.InspectHostPort) []domain.PortMapping {
	result := make([]domain.PortMapping, 0)
	containerPorts := make([]string, 0, len(ports))
	for containerPort := range ports {
		containerPorts = append(containerPorts, containerPort)
	}
	sort.Strings(containerPorts)
	for _, containerPort := range containerPorts {
		bindings := ports[containerPort]
		var protocol string
		var portNumber uint16
		if parts := strings.SplitN(containerPort, "/", 2); len(parts) == 2 {
			protocol = parts[1]
		}
		if parts := strings.SplitN(containerPort, "/", 2); len(parts) > 0 {
			if parsed, err := strconv.ParseUint(parts[0], 10, 16); err == nil {
				portNumber = uint16(parsed)
			}
		}
		for _, binding := range bindings {
			hostPort, _ := strconv.ParseUint(binding.HostPort, 10, 16)
			result = append(result, domain.PortMapping{
				HostIP: binding.HostIP, HostPort: uint16(hostPort), ContainerPort: portNumber, Protocol: protocol,
			})
		}
	}
	return result
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
