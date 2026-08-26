package podman

import (
	"context"
	"errors"
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
	"go.podman.io/podman/v6/pkg/bindings/images"
	imageTypes "go.podman.io/podman/v6/pkg/domain/entities/types"
	"go.podman.io/podman/v6/pkg/inspect"
	"go.podman.io/podman/v6/pkg/specgen"
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
	return &bindingsClient{base: connectionCtx, target: profile.Name}, nil
}

type bindingsClient struct {
	base   context.Context
	target string
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

func (c *bindingsClient) RunContainer(ctx context.Context, request domain.ContainerCreateRequest) (domain.ContainerRunResult, error) {
	if err := request.Validate(); err != nil {
		return domain.ContainerRunResult{}, &domain.OperationError{Category: domain.ErrorInvalidConfig, Action: domain.ActionContainerCreate, TargetID: request.Name, Err: err}
	}
	if err := ctx.Err(); err != nil {
		return domain.ContainerRunResult{}, Wrap(domain.ActionContainerCreate, request.Name, err)
	}
	opCtx, done := c.operationContext(ctx)
	defer done()

	spec := specgen.NewSpecGenerator(request.ImageID, false)
	spec.Name = request.Name
	if len(request.Command) > 0 {
		spec.Command = append([]string(nil), request.Command...)
	}
	terminal := false
	stdin := false
	spec.Terminal = &terminal
	spec.Stdin = &stdin

	response, err := containers.CreateWithSpec(opCtx, spec, nil)
	if err != nil {
		return domain.ContainerRunResult{}, Wrap(domain.ActionContainerCreate, request.Name, err)
	}
	result := domain.ContainerRunResult{
		ContainerID: response.ID,
		Warnings:    append([]string(nil), response.Warnings...),
	}
	if result.ContainerID == "" {
		return result, &domain.OperationError{
			Category: domain.ErrorHost, Action: domain.ActionContainerCreate, TargetID: request.Name,
			Err: errors.New("empty container create response"),
		}
	}
	if err := opCtx.Err(); err != nil {
		return result, partialContainerError(result.ContainerID, err)
	}
	if err := containers.Start(opCtx, result.ContainerID, nil); err != nil {
		return result, partialContainerError(result.ContainerID, Wrap(domain.ActionStart, result.ContainerID, err))
	}
	result.Started = true
	return result, nil
}

func partialContainerError(id string, err error) error {
	return &domain.OperationError{Category: domain.ErrorPartial, Action: domain.ActionStart, TargetID: id, Err: err}
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
	return consumeLogStream(ctx, id, stdoutCh, stderrCh, errCh, emit)
}

func consumeLogStream(
	ctx context.Context,
	id string,
	stdoutCh, stderrCh <-chan string,
	errCh <-chan error,
	emit func(domain.LogLine),
) error {
	for {
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
			if err != nil {
				return Wrap(domain.ActionLogs, id, err)
			}
			// containers.Logs writes every line synchronously before returning.
			// Real Podman connections do not necessarily close the caller-owned
			// channels, so successful completion is the stream terminator.
			return nil
		case <-ctx.Done():
			return Wrap(domain.ActionLogs, id, ctx.Err())
		}
	}
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

func (c *bindingsClient) ListImages(ctx context.Context) ([]domain.ImageSummary, error) {
	if err := ctx.Err(); err != nil {
		return nil, Wrap(domain.ActionImageList, "", err)
	}
	opCtx, done := c.operationContext(ctx)
	defer done()
	all := true
	rows, err := images.List(opCtx, &images.ListOptions{All: &all})
	if err != nil {
		return nil, Wrap(domain.ActionImageList, "", err)
	}
	result := make([]domain.ImageSummary, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		result = append(result, convertImageSummary(row))
	}
	return result, nil
}

func (c *bindingsClient) InspectImage(ctx context.Context, id string) (domain.ImageDetails, error) {
	if err := ctx.Err(); err != nil {
		return domain.ImageDetails{}, Wrap(domain.ActionImageInspect, id, err)
	}
	opCtx, done := c.operationContext(ctx)
	defer done()
	report, err := images.GetImage(opCtx, id, &images.GetOptions{})
	if err != nil {
		return domain.ImageDetails{}, Wrap(domain.ActionImageInspect, id, err)
	}
	if report == nil || report.ImageData == nil {
		return domain.ImageDetails{}, Wrap(domain.ActionImageInspect, id, fmt.Errorf("empty image inspect response"))
	}
	return convertImageDetails(report.ImageData), nil
}

func (c *bindingsClient) PullImage(ctx context.Context, reference string, emit func(domain.ImagePullEvent)) error {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return Wrap(domain.ActionImagePull, reference, errors.New("image reference cannot be empty"))
	}
	if err := ctx.Err(); err != nil {
		return Wrap(domain.ActionImagePull, reference, err)
	}
	opCtx, done := c.operationContext(ctx)
	defer done()
	if emit == nil {
		emit = func(domain.ImagePullEvent) {}
	}
	writer := &imagePullWriter{target: c.target, reference: reference, emit: emit}
	options := (&images.PullOptions{}).
		WithAllTags(false).
		WithQuiet(false).
		WithProgressWriter(writer)
	imageIDs, err := images.Pull(opCtx, reference, options)
	if opCtx.Err() != nil {
		err = opCtx.Err()
	}
	if err != nil {
		kind := domain.ImagePullError
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			kind = domain.ImagePullCancelled
		}
		emit(domain.ImagePullEvent{Target: c.target, Reference: reference, Kind: kind, Text: ErrorMessage(err)})
		return Wrap(domain.ActionImagePull, reference, err)
	}
	emit(domain.ImagePullEvent{Target: c.target, Reference: reference, Kind: domain.ImagePullSuccess, ImageIDs: append([]string(nil), imageIDs...)})
	return nil
}

func (c *bindingsClient) RemoveImage(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return Wrap(domain.ActionImageRemove, id, errors.New("image identity cannot be empty"))
	}
	if err := ctx.Err(); err != nil {
		return Wrap(domain.ActionImageRemove, id, err)
	}
	opCtx, done := c.operationContext(ctx)
	defer done()
	falseValue := false
	report, errs := images.Remove(opCtx, []string{id}, (&images.RemoveOptions{}).
		WithAll(falseValue).
		WithForce(falseValue).
		WithIgnore(falseValue).
		WithLookupManifest(falseValue).
		// NoPrune=true prevents Podman from removing dangling parent images as
		// a side effect of this exact-target operation.
		WithNoPrune(true))
	if opCtx.Err() != nil {
		return Wrap(domain.ActionImageRemove, id, opCtx.Err())
	}
	if len(errs) > 0 {
		return Wrap(domain.ActionImageRemove, id, errors.Join(errs...))
	}
	if report == nil {
		return Wrap(domain.ActionImageRemove, id, fmt.Errorf("empty image removal response"))
	}
	return nil
}

type imagePullWriter struct {
	target    string
	reference string
	emit      func(domain.ImagePullEvent)
}

func (w *imagePullWriter) Write(value []byte) (int, error) {
	if len(value) > 0 {
		w.emit(domain.ImagePullEvent{
			Target: w.target, Reference: w.reference, Kind: domain.ImagePullProgress, Text: string(value),
		})
	}
	return len(value), nil
}

func convertImageSummary(row *imageTypes.ImageSummary) domain.ImageSummary {
	if row == nil {
		return domain.ImageSummary{}
	}
	createdAt := time.Time{}
	if row.Created != 0 {
		createdAt = time.Unix(row.Created, 0)
	}
	return domain.ImageSummary{
		ID:         row.ID,
		References: uniqueStrings(append(append([]string(nil), row.RepoTags...), row.Names...)),
		Digests:    uniqueStrings(row.RepoDigests),
		Digest:     row.Digest,
		Size:       nonNegativeSize(row.Size),
		CreatedAt:  createdAt,
		Containers: row.Containers,
		Dangling:   row.Dangling,
		ReadOnly:   row.ReadOnly,
	}
}

func convertImageDetails(data *inspect.ImageData) domain.ImageDetails {
	if data == nil {
		return domain.ImageDetails{}
	}
	createdAt := time.Time{}
	if data.Created != nil {
		createdAt = *data.Created
	}
	labels := make(map[string]string, len(data.Labels))
	for key, value := range data.Labels {
		labels[key] = value
	}
	return domain.ImageDetails{
		ImageSummary: domain.ImageSummary{
			ID:         data.ID,
			References: uniqueStrings(data.RepoTags),
			Digests:    uniqueStrings(data.RepoDigests),
			Digest:     data.Digest.String(),
			Size:       nonNegativeSize(data.Size),
			CreatedAt:  createdAt,
		},
		ParentID:     data.Parent,
		Architecture: data.Architecture,
		OS:           data.Os,
		Labels:       labels,
	}
}

func uniqueStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func nonNegativeSize(value int64) uint64 {
	if value < 0 {
		return 0
	}
	return uint64(value)
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
