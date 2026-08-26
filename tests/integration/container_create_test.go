//go:build integration

package integration

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Opperiesen/podman-console/internal/domain"
)

func TestLiveContainerCreateWorkflow(t *testing.T) {
	reference := strings.TrimSpace(os.Getenv("PODMAN_CONSOLE_TEST_CREATE_IMAGE"))
	name := strings.TrimSpace(os.Getenv("PODMAN_CONSOLE_TEST_CREATE_NAME"))
	if reference == "" || name == "" {
		t.Skip("PODMAN_CONSOLE_TEST_CREATE_IMAGE and PODMAN_CONSOLE_TEST_CREATE_NAME are not both set")
	}

	client := liveClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	containers, err := client.ListContainers(ctx)
	if err != nil {
		t.Fatalf("list containers before create: %v", err)
	}
	for _, container := range containers {
		if container.Name == name {
			t.Skipf("disposable container name %q already exists; refusing to touch a pre-existing container", name)
		}
	}

	images, err := client.ListImages(ctx)
	if err != nil {
		t.Fatalf("list images before create: %v", err)
	}
	image, ok := findImageReference(images, reference)
	if !ok {
		t.Skipf("local image %q is not available; this workflow never pulls implicitly", reference)
	}
	if image.ID == "" {
		t.Fatal("local image has no full ID")
	}

	command, err := domain.ParseContainerCommand(os.Getenv("PODMAN_CONSOLE_TEST_CREATE_COMMAND"))
	if err != nil {
		t.Fatalf("parse optional create command: %v", err)
	}
	request := domain.ContainerCreateRequest{ImageID: image.ID, ImageReference: reference, Name: name, Command: command}
	if err := request.Validate(); err != nil {
		t.Fatalf("create request is invalid: %v", err)
	}

	createdID := ""
	defer func() {
		if createdID == "" {
			return
		}
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		_ = client.Stop(cleanupCtx, createdID)
		_ = client.Remove(cleanupCtx, createdID)
	}()

	result, err := client.RunContainer(ctx, request)
	createdID = result.ContainerID
	if err != nil {
		t.Fatalf("create/start disposable container: %v (created ID: %q)", err, createdID)
	}
	if createdID == "" || !result.Started {
		t.Fatalf("create/start result = %#v", result)
	}

	details, err := client.InspectContainer(ctx, createdID)
	if err != nil {
		t.Fatalf("inspect created container: %v", err)
	}
	if details.ID != createdID || details.Name != name {
		t.Fatalf("created container identity = id:%q name:%q, want id:%q name:%q", details.ID, details.Name, createdID, name)
	}

	if details.State == domain.StateRunning {
		if err := client.Stop(ctx, createdID); err != nil {
			t.Fatalf("stop created container: %v", err)
		}
	}
	if err := client.Remove(ctx, createdID); err != nil {
		t.Fatalf("remove created container: %v", err)
	}
	createdID = ""

	containers, err = client.ListContainers(ctx)
	if err != nil {
		t.Fatalf("list containers after remove: %v", err)
	}
	for _, container := range containers {
		if container.ID == details.ID || container.Name == name {
			t.Fatalf("disposable container remains after cleanup: %#v", container)
		}
	}
}
