package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/Opperiesen/podman-console/internal/domain"
	"github.com/Opperiesen/podman-console/internal/ui"
	"github.com/Opperiesen/podman-console/tests/fixtures"
)

func testCreateImage() domain.ImageSummary {
	return domain.ImageSummary{
		ID:         "sha256:full-image-id-1234567890",
		References: []string{"quay.io/example/app:latest"},
	}
}

func TestContainerCreateFormCapturesExactLocalImageWithoutMutation(t *testing.T) {
	model, client := testModel(t)
	model.screen = ui.ScreenImages
	model.images = []domain.ImageSummary{testCreateImage()}
	client.Images = []domain.ImageSummary{testCreateImage()}
	model.generation = 7
	updated, cmd := model.Update(keyPress("n", 'n'))
	model = updated.(*Model)
	if cmd == nil || model.screen != ui.ScreenContainerCreate {
		t.Fatalf("open create form = screen:%q command:%v", model.screen, cmd)
	}
	model.containerCreateInputs[0].SetValue("web-prod")
	model.containerCreateInputs[1].SetValue("sleep 60")
	updated, cmd = model.Update(keyPress("", tea.KeyEnter))
	model = updated.(*Model)
	if cmd != nil || model.mode != ui.ModeConfirm {
		t.Fatalf("submit create form = mode:%q command:%v", model.mode, cmd)
	}
	if got := model.pendingContainerCreate; got.ImageID != testCreateImage().ID || got.ImageReference != "quay.io/example/app:latest" || got.Name != "web-prod" || strings.Join(got.Command, " ") != "sleep 60" {
		t.Fatalf("captured request = %#v", got)
	}
	if len(client.Calls) != 0 {
		t.Fatalf("form contacted Podman: %v", client.Calls)
	}
	if !strings.Contains(model.View().Content, "sha256:full-image-id-1234567890") || !strings.Contains(model.View().Content, "web-prod") || !strings.Contains(model.View().Content, "sleep 60") {
		t.Fatalf("confirmation does not show exact request:\n%s", model.View().Content)
	}
}

func TestContainerCreateConfirmationCancellationSendsNoMutation(t *testing.T) {
	model, client := testModel(t)
	model.screen = ui.ScreenImages
	model.images = []domain.ImageSummary{testCreateImage()}
	model.generation = 3
	_ = model.openContainerCreate()
	model.containerCreateInputs[0].SetValue("web")
	updated, cmd := model.Update(keyPress("", tea.KeyEnter))
	model = updated.(*Model)
	if cmd != nil || model.mode != ui.ModeConfirm {
		t.Fatalf("confirmation state = mode:%q command:%v", model.mode, cmd)
	}
	updated, cmd = model.Update(keyPress("", tea.KeyEscape))
	model = updated.(*Model)
	if cmd != nil || model.mode != ui.ModeNormal || len(client.Calls) != 0 {
		t.Fatalf("cancelled create = mode:%q calls:%v command:%v", model.mode, client.Calls, cmd)
	}
	if !strings.Contains(model.status, "aucune mutation") {
		t.Fatalf("cancel status = %q", model.status)
	}
}

func TestContainerCreateSuccessUsesExactCreateStartOrderAndRefreshesBothInventories(t *testing.T) {
	model, client := testModel(t)
	model.screen = ui.ScreenImages
	model.images = []domain.ImageSummary{testCreateImage()}
	client.Images = []domain.ImageSummary{testCreateImage()}
	model.generation = 7
	_ = model.openContainerCreate()
	model.containerCreateInputs[0].SetValue("web-prod")
	model.containerCreateInputs[1].SetValue("sleep 60")
	updated, _ := model.Update(keyPress("", tea.KeyEnter))
	model = updated.(*Model)
	updated, cmd := model.Update(keyPress("", tea.KeyEnter))
	model = updated.(*Model)
	if cmd == nil {
		t.Fatal("confirmation did not start create command")
	}
	updated, cmd = model.Update(cmd())
	model = updated.(*Model)
	if cmd == nil || model.containerCreateStatus != domain.ContainerCreateRefreshing {
		t.Fatalf("run result = status:%q command:%v", model.containerCreateStatus, cmd)
	}
	updated, cmd = model.Update(cmd())
	model = updated.(*Model)
	if cmd != nil || model.screen != ui.ScreenInventory || model.containerCreateStatus != domain.ContainerCreateSucceeded {
		t.Fatalf("refresh result = screen:%q status:%q command:%v", model.screen, model.containerCreateStatus, cmd)
	}
	if len(client.Calls) != 4 || client.Calls[0] != "container_create:sha256:full-image-id-1234567890:web-prod" || client.Calls[1] != "start:created-container" || client.Calls[2] != string(domain.ActionList) || client.Calls[3] != string(domain.ActionImageList) {
		t.Fatalf("Podman call order = %v", client.Calls)
	}
	if len(model.containers) != 2 || model.containers[1].Name != "web-prod" || model.selectedID() != "created-container" {
		t.Fatalf("refreshed containers = selected:%q containers:%#v", model.selectedID(), model.containers)
	}
	if len(model.images) != 1 || model.images[0].ID != testCreateImage().ID {
		t.Fatalf("refreshed images = %#v", model.images)
	}
	if !strings.Contains(model.status, "inventaires actualisés") {
		t.Fatalf("success status = %q", model.status)
	}
}

func TestContainerCreateStartFailurePreservesCreatedIDAndRefreshesWithoutRemove(t *testing.T) {
	model, client := testModel(t)
	model.screen = ui.ScreenImages
	model.images = []domain.ImageSummary{testCreateImage()}
	model.generation = 7
	client.RunResult = domain.ContainerRunResult{ContainerID: "created-after-start-failure"}
	client.Errors[domain.ActionStart] = errors.New("permission denied")
	_ = model.openContainerCreate()
	model.containerCreateInputs[0].SetValue("web-partial")
	updated, _ := model.Update(keyPress("", tea.KeyEnter))
	model = updated.(*Model)
	updated, cmd := model.Update(keyPress("", tea.KeyEnter))
	model = updated.(*Model)
	updated, cmd = model.Update(cmd())
	model = updated.(*Model)
	if cmd == nil || model.containerCreateStatus != domain.ContainerCreatePartial || model.containerCreateResult.ContainerID != "created-after-start-failure" {
		t.Fatalf("partial run = status:%q result:%#v command:%v", model.containerCreateStatus, model.containerCreateResult, cmd)
	}
	updated, cmd = model.Update(cmd())
	model = updated.(*Model)
	if cmd != nil || !strings.Contains(model.status, "created-after-start-failure") {
		t.Fatalf("partial refresh = status:%q command:%v", model.status, cmd)
	}
	for _, call := range client.Calls {
		if strings.HasPrefix(call, "remove:") {
			t.Fatalf("partial operation removed the container: %v", client.Calls)
		}
	}
}

func TestContainerCreateRejectsStaleImageBeforeConfirmation(t *testing.T) {
	model, client := testModel(t)
	model.screen = ui.ScreenImages
	model.images = []domain.ImageSummary{testCreateImage()}
	model.generation = 4
	_ = model.openContainerCreate()
	model.containerCreateInputs[0].SetValue("web")
	model.generation++
	updated, cmd := model.Update(keyPress("", tea.KeyEnter))
	model = updated.(*Model)
	if cmd != nil || model.mode == ui.ModeConfirm || len(client.Calls) != 0 {
		t.Fatalf("stale image was accepted = mode:%q command:%v calls:%v", model.mode, cmd, client.Calls)
	}
	if !strings.Contains(model.containerCreateError.Error(), "plus disponible") {
		t.Fatalf("stale image error = %v", model.containerCreateError)
	}
}

func TestContainerCreateRejectsInvalidInputWithoutMutation(t *testing.T) {
	model, client := testModel(t)
	model.screen = ui.ScreenImages
	model.images = []domain.ImageSummary{testCreateImage()}
	model.generation = 4
	_ = model.openContainerCreate()
	model.containerCreateInputs[0].SetValue("web name")
	model.containerCreateInputs[1].SetValue("sh -c 'sleep 60'")
	updated, cmd := model.Update(keyPress("", tea.KeyEnter))
	model = updated.(*Model)
	if cmd != nil || model.mode == ui.ModeConfirm || len(client.Calls) != 0 {
		t.Fatalf("invalid form was accepted = mode:%q command:%v calls:%v", model.mode, cmd, client.Calls)
	}
	if model.containerCreateError == nil || !strings.Contains(model.containerCreateError.Error(), "shell") && !strings.Contains(model.containerCreateError.Error(), "container name") {
		t.Fatalf("invalid form error = %v", model.containerCreateError)
	}
}

func TestContainerCreateDuplicateSubmitDoesNotStartSecondRequest(t *testing.T) {
	model, client := testModel(t)
	model.screen = ui.ScreenImages
	model.images = []domain.ImageSummary{testCreateImage()}
	model.generation = 4
	_ = model.openContainerCreate()
	model.containerCreateInputs[0].SetValue("web")
	updated, _ := model.Update(keyPress("", tea.KeyEnter))
	model = updated.(*Model)
	updated, cmd := model.Update(keyPress("", tea.KeyEnter))
	model = updated.(*Model)
	if cmd == nil {
		t.Fatal("confirmation did not start create request")
	}
	updated, second := model.Update(keyPress("", tea.KeyEnter))
	model = updated.(*Model)
	if second != nil || len(client.Calls) != 0 {
		t.Fatalf("duplicate submit started a second command = command:%v calls:%v", second, client.Calls)
	}
	_ = cmd()
}

func TestContainerCreateCancellationAfterCreateKeepsPartialID(t *testing.T) {
	model, _ := testModel(t)
	model.client = &fixtures.Client{Errors: map[domain.Action]error{}}
	model.images = []domain.ImageSummary{testCreateImage()}
	model.containerCreateRequest = domain.ContainerCreateRequest{ImageID: testCreateImage().ID, ImageReference: "quay.io/example/app:latest", Name: "web"}
	model.containerCreateTarget = model.connectionIdentity()
	model.containerCreateGeneration = model.generation
	model.containerCreateRunning = true
	updated, cmd := model.Update(ContainerRunFinishedMsg{
		Generation: model.generation, Target: model.containerCreateTarget, Request: model.containerCreateRequest,
		Result: domain.ContainerRunResult{ContainerID: "created-before-cancel"}, Err: context.Canceled,
	})
	model = updated.(*Model)
	if cmd == nil || model.containerCreateStatus != domain.ContainerCreatePartial || model.containerCreateResult.ContainerID != "created-before-cancel" {
		t.Fatalf("cancelled partial result = status:%q result:%#v command:%v", model.containerCreateStatus, model.containerCreateResult, cmd)
	}
}

func TestContainerCreateCancellationBeforeCreateReportsNoMutation(t *testing.T) {
	model, _ := testModel(t)
	model.containerCreateRequest = domain.ContainerCreateRequest{ImageID: testCreateImage().ID, Name: "web"}
	model.containerCreateTarget = model.connectionIdentity()
	model.containerCreateGeneration = model.generation
	model.containerCreateRunning = true
	updated, cmd := model.Update(ContainerRunFinishedMsg{
		Generation: model.generation, Target: model.containerCreateTarget, Request: model.containerCreateRequest,
		Err: context.Canceled,
	})
	model = updated.(*Model)
	if cmd != nil || model.containerCreateStatus != domain.ContainerCreateCancelled || model.containerCreateResult.ContainerID != "" {
		t.Fatalf("cancelled create = status:%q result:%#v command:%v", model.containerCreateStatus, model.containerCreateResult, cmd)
	}
	if !strings.Contains(model.status, "aucune création") || model.err != nil {
		t.Fatalf("cancelled create feedback = status:%q err:%v", model.status, model.err)
	}
}
