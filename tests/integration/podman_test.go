//go:build integration

package integration

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Opperiesen/podman-console/internal/domain"
	"github.com/Opperiesen/podman-console/internal/podman"
)

func TestLiveInventory(t *testing.T) {
	client := liveClient(t)
	if _, err := client.ListContainers(context.Background()); err != nil {
		t.Fatalf("list containers: %v", err)
	}
}

func TestLiveContainerWorkflow(t *testing.T) {
	selector := os.Getenv("PODMAN_CONSOLE_TEST_CONTAINER")
	if selector == "" {
		t.Skip("PODMAN_CONSOLE_TEST_CONTAINER is not set")
	}

	client := liveClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	rows, err := client.ListContainers(ctx)
	if err != nil {
		t.Fatalf("list containers: %v", err)
	}
	targetID := ""
	for _, row := range rows {
		if row.Name == selector || row.ID == selector || strings.HasPrefix(row.ID, selector) {
			targetID = row.ID
			break
		}
	}
	if targetID == "" {
		t.Fatalf("test container %q not found", selector)
	}

	removed := false
	defer func() {
		if removed {
			return
		}
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 20*time.Second)
		_ = client.Stop(stopCtx, targetID)
		stopCancel()
		removeCtx, removeCancel := context.WithTimeout(context.Background(), 10*time.Second)
		_ = client.Remove(removeCtx, targetID)
		removeCancel()
	}()

	details, err := client.InspectContainer(ctx, targetID)
	if err != nil {
		t.Fatalf("inspect container: %v", err)
	}
	if details.ID != targetID {
		t.Fatalf("inspect returned ID %q, want %q", details.ID, targetID)
	}

	var logs []domain.LogLine
	logsCtx, logsCancel := context.WithTimeout(ctx, 10*time.Second)
	err = client.StreamLogs(logsCtx, targetID, podman.LogOptions{Tail: 20}, func(line domain.LogLine) {
		logs = append(logs, line)
	})
	logsCancel()
	if err != nil {
		t.Fatalf("stream logs: %v", err)
	}
	if len(logs) == 0 {
		t.Fatal("stream logs returned no lines")
	}

	statsCtx, statsCancel := context.WithTimeout(ctx, 10*time.Second)
	var samples []domain.ContainerStats
	err = client.StreamStats(statsCtx, targetID, func(sample domain.ContainerStats) {
		samples = append(samples, sample)
		statsCancel()
	})
	statsCancel()
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("stream stats: %v", err)
	}
	if len(samples) == 0 {
		t.Fatal("stream stats returned no samples")
	}

	if err := client.Restart(ctx, targetID); err != nil {
		t.Fatalf("restart container: %v", err)
	}
	if err := client.Stop(ctx, targetID); err != nil {
		t.Fatalf("stop container: %v", err)
	}
	if err := client.Start(ctx, targetID); err != nil {
		t.Fatalf("start container: %v", err)
	}
	running, err := client.InspectContainer(ctx, targetID)
	if err != nil {
		t.Fatalf("inspect restarted container: %v", err)
	}
	if running.State != domain.StateRunning {
		t.Fatalf("container state after start = %q, want %q", running.State, domain.StateRunning)
	}
	if err := client.Stop(ctx, targetID); err != nil {
		t.Fatalf("final stop container: %v", err)
	}
	if err := client.Remove(ctx, targetID); err != nil {
		t.Fatalf("remove container: %v", err)
	}
	removed = true

	rows, err = client.ListContainers(ctx)
	if err != nil {
		t.Fatalf("list containers after remove: %v", err)
	}
	for _, row := range rows {
		if row.ID == targetID {
			t.Fatalf("removed container %q is still listed", targetID)
		}
	}
}

func liveClient(t *testing.T) podman.Client {
	t.Helper()
	uri := os.Getenv("PODMAN_CONSOLE_URI")
	if uri == "" {
		t.Skip("PODMAN_CONSOLE_URI is not set")
	}
	profile := domain.ConnectionProfile{Name: "integration", URI: uri, IdentityPath: os.Getenv("PODMAN_CONSOLE_IDENTITY")}
	client, err := (podman.BindingsFactory{}).Connect(context.Background(), profile)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	return client
}
