package podman

import (
	"context"
	"time"

	"github.com/Opperiesen/podman-console/internal/domain"
)

type LogOptions struct {
	Follow     bool
	Tail       int
	Timestamps bool
}

type Client interface {
	ListContainers(ctx context.Context) ([]domain.ContainerSummary, error)
	InspectContainer(ctx context.Context, id string) (domain.ContainerDetails, error)
	Start(ctx context.Context, id string) error
	Stop(ctx context.Context, id string) error
	Restart(ctx context.Context, id string) error
	Remove(ctx context.Context, id string) error
	StreamLogs(ctx context.Context, id string, options LogOptions, emit func(domain.LogLine)) error
	StreamStats(ctx context.Context, id string, emit func(domain.ContainerStats)) error
}

type Factory interface {
	Connect(ctx context.Context, profile domain.ConnectionProfile) (Client, error)
}

type StatsInterval time.Duration

const DefaultStatsInterval StatsInterval = StatsInterval(2 * time.Second)
