package ui

import (
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/help"
	"github.com/Opperiesen/podman-console/internal/domain"
)

func TestRenderImageInventoryIncludesMetadataAndExplicitEmptyValues(t *testing.T) {
	image := domain.ImageSummary{
		ID:         "sha256:abcdef0123456789",
		References: []string{"registry.example/team/very-long-application:production"},
		Digests:    []string{"registry.example/team/app@sha256:1234567890abcdef"},
		Size:       1024 * 1024,
		CreatedAt:  time.Unix(1_700_000_000, 0),
		Dangling:   true,
	}
	output := Render(ViewData{
		Width: 40, Height: 16, Screen: ScreenImages, Mode: ModeNormal,
		Profile: domain.ConnectionProfile{Name: "local", URI: "unix:///run/podman.sock"}, Connected: true,
		Images: []domain.ImageSummary{image, {ID: "sha256:empty"}}, ImageSelected: 1,
		Keys: NewKeyMap(), Help: help.New(),
	})
	for _, want := range []string{"IMAGES", "sha256:empty", "dangling", "—", "local"} {
		if !strings.Contains(output, want) {
			t.Errorf("image inventory does not contain %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, "registry.example/team/very-long-application:production") {
		t.Fatal("narrow image inventory did not truncate a long reference")
	}
}

func TestRenderImageDetailsIncludesAllIdentityFields(t *testing.T) {
	output := Render(ViewData{
		Width: 100, Height: 30, Screen: ScreenImageDetails, Mode: ModeNormal,
		Profile: domain.ConnectionProfile{Name: "remote"}, Connected: true,
		ImageDetails: &domain.ImageDetails{
			ImageSummary: domain.ImageSummary{
				ID: "sha256:full-image-id", References: []string{"app:latest", "app:stable"},
				Digests: []string{"app@sha256:abc", "app@sha256:def"}, Size: 2048, Containers: 4,
				CreatedAt: time.Unix(1_700_000_000, 0),
			},
			ParentID: "sha256:parent", Architecture: "arm64", OS: "linux", Labels: map[string]string{"team": "platform"},
		}, Keys: NewKeyMap(), Help: help.New(),
	})
	for _, want := range []string{"sha256:full-image-id", "app:latest", "app:stable", "app@sha256:abc", "app@sha256:def", "team=platform", "Conteneurs     4", "arm64"} {
		if !strings.Contains(output, want) {
			t.Errorf("image details does not contain %q:\n%s", want, output)
		}
	}
}

func TestRenderImagePullPreservesOrderedProgressAndStoppedFeedback(t *testing.T) {
	output := Render(ViewData{
		Width: 80, Height: 20, Screen: ScreenImagePull, Mode: ModeNormal,
		Profile: domain.ConnectionProfile{Name: "local"}, Connected: true,
		ImagePullReference: "quay.io/example/app:latest", ImagePullStatus: domain.ImageOperationCancelled,
		ImagePullStopped: true, ImagePullEvents: []domain.ImagePullEvent{
			{Kind: domain.ImagePullProgress, Text: "first layer"},
			{Kind: domain.ImagePullProgress, Text: "second layer"},
		}, Keys: NewKeyMap(), Help: help.New(),
	})
	if strings.Index(output, "first layer") > strings.Index(output, "second layer") || !strings.Contains(output, "flux arrêté") || !strings.Contains(output, "quay.io/example/app:latest") {
		t.Fatalf("pull rendering lost order or stopped state:\n%s", output)
	}
}

func TestRenderImageConfirmationNamesHostReferenceAndID(t *testing.T) {
	output := Render(ViewData{
		Width: 90, Height: 24, Screen: ScreenImages, Mode: ModeConfirm,
		Profile:       domain.ConnectionProfile{Name: "rocky", URI: "ssh://rocky.example/run/podman.sock"},
		ConfirmAction: "Suppression", ConfirmResource: "image", ConfirmTarget: "app:latest", ConfirmTargetID: "sha256:full-image-id",
		Keys: NewKeyMap(), Help: help.New(),
	})
	for _, want := range []string{"rocky", "app:latest", "sha256:full-image-id", "l’image"} {
		if !strings.Contains(output, want) {
			t.Errorf("image confirmation does not contain %q:\n%s", want, output)
		}
	}
}
