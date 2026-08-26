package ui

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/help"
	"github.com/Opperiesen/podman-console/internal/domain"
)

func TestRenderContainerCreateFormKeepsImageIdentityReadOnly(t *testing.T) {
	output := Render(ViewData{
		Width: 100, Height: 30, Screen: ScreenContainerCreate, Mode: ModeNormal,
		Profile: domain.ConnectionProfile{Name: "rocky", URI: "ssh://rocky.example/run/podman.sock"}, Connected: true,
		ContainerCreateImageReference: "quay.io/example/app:latest",
		ContainerCreateImageID:        "sha256:full-image-id-1234567890",
		ContainerCreateFields:         []string{"web-prod", "sleep 60"}, ContainerCreateFocus: 0,
		Keys: NewKeyMap(), Help: help.New(),
	})
	for _, want := range []string{"NOUVEAU CONTENEUR", "rocky", "quay.io/example/app:latest", "sha256:full-image-id-1234567890", "web-prod", "sleep 60", "aucun shell"} {
		if !strings.Contains(output, want) {
			t.Errorf("container create form does not contain %q:\n%s", want, output)
		}
	}
}

func TestRenderContainerCreateConfirmationNamesExactTargetAndDefaultCommand(t *testing.T) {
	output := Render(ViewData{
		Width: 100, Height: 30, Screen: ScreenContainerCreate, Mode: ModeConfirm,
		Profile: domain.ConnectionProfile{Name: "rocky", URI: "ssh://rocky.example/run/podman.sock"}, Connected: true,
		ConfirmAction: "Création de conteneur", ConfirmTarget: "web-prod", ConfirmTargetID: "sha256:full-image-id-1234567890", ConfirmResource: "container_create",
		ContainerCreateRequest: domain.ContainerCreateRequest{
			ImageID: "sha256:full-image-id-1234567890", ImageReference: "quay.io/example/app:latest", Name: "web-prod",
		},
		Keys: NewKeyMap(), Help: help.New(),
	})
	for _, want := range []string{"rocky", "quay.io/example/app:latest", "sha256:full-image-id-1234567890", "web-prod", "commande par défaut", "aucun pull implicite"} {
		if !strings.Contains(output, want) {
			t.Errorf("container create confirmation does not contain %q:\n%s", want, output)
		}
	}
}

func TestRenderContainerCreatePartialResultShowsExactID(t *testing.T) {
	output := Render(ViewData{
		Width: 38, Height: 16, Screen: ScreenInventory, Mode: ModeNormal,
		Profile: domain.ConnectionProfile{Name: "rocky"}, Connected: true,
		Status:                "Conteneur créé mais non démarré · ID created-after-start-failure · inventaires actualisés",
		ContainerCreateResult: domain.ContainerRunResult{ContainerID: "created-after-start-failure"},
		Keys:                  NewKeyMap(), Help: help.New(),
	})
	if !strings.Contains(output, "created-after-start-failure") || !strings.Contains(output, "non démarré") {
		t.Fatalf("partial result lost exact ID or actionable error:\n%s", output)
	}
}
