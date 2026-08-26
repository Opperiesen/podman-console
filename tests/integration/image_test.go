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

func TestLiveImageWorkflow(t *testing.T) {
	reference := strings.TrimSpace(os.Getenv("PODMAN_CONSOLE_TEST_IMAGE"))
	if reference == "" {
		t.Skip("PODMAN_CONSOLE_TEST_IMAGE is not set")
	}

	client := liveClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	before, err := client.ListImages(ctx)
	if err != nil {
		t.Fatalf("list images before pull: %v", err)
	}
	if image, ok := findImageReference(before, reference); ok {
		t.Skipf("disposable image reference %q already exists as %q; refusing to remove a pre-existing image", reference, image.ID)
	}

	cleanup := true
	defer func() {
		if !cleanup {
			return
		}
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		if images, listErr := client.ListImages(cleanupCtx); listErr == nil {
			if image, ok := findImageReference(images, reference); ok {
				_ = client.RemoveImage(cleanupCtx, image.ID)
			}
		}
	}()

	var events []domain.ImagePullEvent
	err = client.PullImage(ctx, reference, func(event domain.ImagePullEvent) {
		events = append(events, event)
	})
	if err != nil {
		t.Fatalf("pull image %q: %v", reference, err)
	}

	afterPull, err := client.ListImages(ctx)
	if err != nil {
		t.Fatalf("list images after pull: %v", err)
	}
	pulled, ok := findImageReference(afterPull, reference)
	if !ok {
		t.Fatalf("pulled image %q was not returned by authoritative inventory; events: %#v", reference, events)
	}
	if pulled.ID == "" {
		t.Fatal("pulled image has no ID")
	}
	details, err := client.InspectImage(ctx, pulled.ID)
	if err != nil {
		t.Fatalf("inspect pulled image: %v", err)
	}
	if details.ID != pulled.ID {
		t.Fatalf("inspect image ID = %q, want %q", details.ID, pulled.ID)
	}
	if err := client.RemoveImage(ctx, pulled.ID); err != nil {
		t.Fatalf("remove pulled image: %v", err)
	}
	cleanup = false

	afterRemove, err := client.ListImages(ctx)
	if err != nil {
		t.Fatalf("list images after remove: %v", err)
	}
	if image, ok := findImageReference(afterRemove, reference); ok {
		t.Fatalf("removed disposable image is still listed as %q", image.ID)
	}
}

func findImageReference(images []domain.ImageSummary, reference string) (domain.ImageSummary, bool) {
	for _, image := range images {
		for _, candidate := range image.References {
			if candidate == reference {
				return image, true
			}
		}
	}
	return domain.ImageSummary{}, false
}
