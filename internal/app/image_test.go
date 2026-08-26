package app

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/Opperiesen/podman-console/internal/domain"
	"github.com/Opperiesen/podman-console/internal/ui"
)

func testImage() domain.ImageSummary {
	return domain.ImageSummary{
		ID:         "sha256:image-abcdef0123456789",
		References: []string{"registry.example/team/web:latest"},
		Digests:    []string{"registry.example/team/web@sha256:1234567890abcdef"},
		Size:       12 * 1024,
		CreatedAt:  time.Unix(1_700_000_000, 0),
		Containers: 2,
	}
}

func TestModelLoadsImagesFiltersLocallyAndKeepsEmptyState(t *testing.T) {
	model, client := testModel(t)
	client.Images = []domain.ImageSummary{testImage(), {ID: "sha256:database", References: []string{"postgres:16"}}}

	updated, cmd := model.Update(keyPress("i", 'i'))
	model = updated.(*Model)
	if model.screen != ui.ScreenImages || cmd == nil {
		t.Fatalf("opening images = screen:%q command:%v", model.screen, cmd)
	}
	updated, cmd = model.Update(cmd())
	model = updated.(*Model)
	if cmd != nil || len(model.images) != 2 {
		t.Fatalf("image inventory = %#v command:%v", model.images, cmd)
	}

	model.imageFilter = "123456"
	visible := model.visibleImages()
	if len(visible) != 1 || visible[0].ID != client.Images[0].ID {
		t.Fatalf("filtered images = %#v", visible)
	}

	model.images = nil
	output := model.View().Content
	if !strings.Contains(output, "Aucune image") || !strings.Contains(output, "P") {
		t.Fatalf("empty image view = %s", output)
	}
}

func TestModelLoadsImageDetailsAndIgnoresStaleSelection(t *testing.T) {
	model, client := testModel(t)
	image := testImage()
	client.Images = []domain.ImageSummary{image}
	client.ImageDetails = map[string]domain.ImageDetails{
		image.ID: {
			ImageSummary: image,
			Labels:       map[string]string{"org.example.role": "web"},
			Architecture: "arm64",
			OS:           "linux",
		},
	}
	model.images = client.Images
	model.screen = ui.ScreenImages

	updated, cmd := model.Update(keyPress("", tea.KeyEnter))
	model = updated.(*Model)
	if cmd == nil || model.screen != ui.ScreenImageDetails {
		t.Fatalf("open image details = screen:%q command:%v", model.screen, cmd)
	}
	updated, cmd = model.Update(cmd())
	model = updated.(*Model)
	if cmd != nil || model.imageDetails == nil || model.imageDetails.Labels["org.example.role"] != "web" {
		t.Fatalf("image details = %#v command:%v", model.imageDetails, cmd)
	}

	model.imageDetails = nil
	model.imageDetailTarget = image.ID
	model.generation = 8
	updated, cmd = model.Update(ImageDetailsLoadedMsg{Generation: 7, TargetID: image.ID, Details: client.ImageDetails[image.ID]})
	model = updated.(*Model)
	if cmd != nil || model.imageDetails != nil {
		t.Fatal("stale image detail response replaced current state")
	}
}

func TestModelPullPreservesArrivalOrderAndRefreshesAuthoritatively(t *testing.T) {
	model, client := testModel(t)
	client.PullEvents = []domain.ImagePullEvent{
		{Kind: domain.ImagePullProgress, Text: "layer-a\n"},
		{Kind: domain.ImagePullProgress, Text: "layer-b\n"},
		{Kind: domain.ImagePullSuccess, ImageIDs: []string{"sha256:pulled"}},
	}
	model.screen = ui.ScreenImages
	model.imagePullInput.SetValue("quay.io/example/web:latest")

	updated, cmd := model.Update(keyPress("P", 'P'))
	model = updated.(*Model)
	if cmd == nil || model.screen != ui.ScreenImagePull {
		t.Fatalf("open pull = screen:%q command:%v", model.screen, cmd)
	}
	model.imagePullInput.SetValue("quay.io/example/web:latest")
	updated, cmd = model.Update(keyPress("", tea.KeyEnter))
	model = updated.(*Model)
	if cmd == nil || !model.imagePulling {
		t.Fatal("pull did not start")
	}

	for steps := 0; cmd != nil && steps < 12; steps++ {
		updated, cmd = model.Update(cmd())
		model = updated.(*Model)
	}
	if cmd != nil {
		t.Fatal("pull command did not terminate")
	}
	if len(model.imagePullEvents) != 3 || model.imagePullEvents[0].Text != "layer-a\n" || model.imagePullEvents[1].Text != "layer-b\n" {
		t.Fatalf("pull event order = %#v", model.imagePullEvents)
	}
	if model.imagePullStatus != domain.ImageOperationSucceeded || model.screen != ui.ScreenImages {
		t.Fatalf("pull outcome = status:%q screen:%q", model.imagePullStatus, model.screen)
	}
	if !containsCall(client.Calls, string(domain.ActionImageList)) {
		t.Fatalf("successful pull did not refresh images: %v", client.Calls)
	}
}

func TestModelPullCancellationStopsRequestAndRefreshes(t *testing.T) {
	model, client := testModel(t)
	wait := make(chan struct{})
	client.PullWait = wait
	model.screen = ui.ScreenImages
	model.imagePullInput.SetValue("quay.io/example/web:latest")

	updated, cmd := model.Update(keyPress("P", 'P'))
	model = updated.(*Model)
	model.imagePullInput.SetValue("quay.io/example/web:latest")
	updated, cmd = model.Update(keyPress("", tea.KeyEnter))
	model = updated.(*Model)
	if cmd == nil || !model.imagePulling {
		t.Fatal("pull did not start")
	}
	updated, cmd = model.Update(keyPress("", tea.KeyEscape))
	model = updated.(*Model)
	if cmd == nil || model.imagePulling || model.imagePullStatus != domain.ImageOperationCancelled {
		t.Fatalf("cancelled pull = pulling:%v status:%q command:%v", model.imagePulling, model.imagePullStatus, cmd)
	}
	updated, cmd = model.Update(cmd())
	model = updated.(*Model)
	close(wait)
	if cmd != nil || !containsCall(client.Calls, string(domain.ActionImageList)) {
		t.Fatalf("cancel refresh = command:%v calls:%v", cmd, client.Calls)
	}
}

func TestModelImageRemovalRequiresExactConfirmationAndRefreshes(t *testing.T) {
	model, client := testModel(t)
	image := testImage()
	client.Images = []domain.ImageSummary{image}
	client.ImageDetails = map[string]domain.ImageDetails{image.ID: {ImageSummary: image}}
	model.images = client.Images
	model.screen = ui.ScreenImages

	updated, cmd := model.Update(keyPress("D", 'D'))
	model = updated.(*Model)
	if cmd != nil || model.mode != ui.ModeConfirm || model.pendingID != image.ID || len(client.Calls) != 0 {
		t.Fatalf("image removal confirmation = mode:%q target:%q calls:%v", model.mode, model.pendingID, client.Calls)
	}
	updated, cmd = model.Update(keyPress("n", 'n'))
	model = updated.(*Model)
	if cmd != nil || model.mode != ui.ModeNormal || len(client.Calls) != 0 {
		t.Fatalf("cancelled image removal = mode:%q calls:%v", model.mode, client.Calls)
	}

	updated, cmd = model.Update(keyPress("D", 'D'))
	model = updated.(*Model)
	updated, cmd = model.Update(keyPress("y", 'y'))
	model = updated.(*Model)
	if cmd == nil {
		t.Fatal("confirmed image removal did not start")
	}
	updated, cmd = model.Update(cmd())
	model = updated.(*Model)
	if cmd == nil {
		t.Fatal("successful image removal did not refresh")
	}
	updated, cmd = model.Update(cmd())
	model = updated.(*Model)
	if cmd != nil || len(model.images) != 0 || !containsCall(client.Calls, string(domain.ActionImageRemove)+":"+image.ID) {
		t.Fatalf("image removal outcome = images:%#v calls:%v command:%v", model.images, client.Calls, cmd)
	}
}

func TestModelImageRemovalRejectsStaleConfirmation(t *testing.T) {
	model, client := testModel(t)
	image := testImage()
	model.images = []domain.ImageSummary{image}
	model.screen = ui.ScreenImages
	updated, cmd := model.Update(keyPress("D", 'D'))
	model = updated.(*Model)
	if cmd != nil {
		t.Fatal("removal confirmation unexpectedly returned a command")
	}
	model.generation++
	updated, cmd = model.Update(keyPress("y", 'y'))
	model = updated.(*Model)
	if cmd != nil || len(client.Calls) != 0 || model.mode != ui.ModeNormal {
		t.Fatalf("stale confirmation mutated target: mode:%q calls:%v command:%v", model.mode, client.Calls, cmd)
	}
}

func TestModelImagePullFailurePreservesProgressAndActionableError(t *testing.T) {
	model, client := testModel(t)
	client.PullEvents = []domain.ImagePullEvent{{Kind: domain.ImagePullProgress, Text: "layer-a\n"}}
	client.PullErr = errors.New("manifest unknown")
	model.screen = ui.ScreenImagePull
	model.imagePullInput.SetValue("quay.io/example/missing:latest")
	updated, cmd := model.Update(keyPress("", tea.KeyEnter))
	model = updated.(*Model)
	for steps := 0; cmd != nil && steps < 8; steps++ {
		updated, cmd = model.Update(cmd())
		model = updated.(*Model)
	}
	if model.imagePullStatus != domain.ImageOperationFailed || len(model.imagePullEvents) != 1 || model.imagePullError == nil || !strings.Contains(model.imagePullError.Error(), "Registre") {
		t.Fatalf("pull failure = status:%q events:%#v error:%v", model.imagePullStatus, model.imagePullEvents, model.imagePullError)
	}
}

func TestModelIgnoresPullEventsFromAnotherConnection(t *testing.T) {
	model, _ := testModel(t)
	model.screen = ui.ScreenImagePull
	model.imagePullTarget = model.connectionIdentity()
	model.imagePullGeneration = 5
	model.imagePulling = true
	updated, cmd := model.Update(imagePullStreamEvent{
		Generation: 5, Target: "another-host", Event: &domain.ImagePullEvent{Kind: domain.ImagePullProgress, Text: "wrong host"},
	})
	model = updated.(*Model)
	if cmd != nil || len(model.imagePullEvents) != 0 || !model.imagePulling {
		t.Fatalf("stale pull event changed state: events:%#v pulling:%v command:%v", model.imagePullEvents, model.imagePulling, cmd)
	}
}

func TestModelImageRemovalMapsInUseAndAuthorizationErrors(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		want string
	}{
		{name: "in use", err: errors.New("image is in use by a container"), want: "Image utilisée"},
		{name: "authorization", err: errors.New("permission denied"), want: "Autorisation refusée"},
	} {
		t.Run(test.name, func(t *testing.T) {
			model, client := testModel(t)
			image := testImage()
			client.Images = []domain.ImageSummary{image}
			model.images = client.Images
			model.screen = ui.ScreenImages
			client.Errors[domain.ActionImageRemove] = test.err

			updated, cmd := model.Update(keyPress("D", 'D'))
			model = updated.(*Model)
			updated, cmd = model.Update(keyPress("y", 'y'))
			model = updated.(*Model)
			if cmd == nil {
				t.Fatal("image removal did not start")
			}
			updated, cmd = model.Update(cmd())
			model = updated.(*Model)
			if cmd != nil || model.err == nil || !strings.Contains(model.err.Error(), test.want) {
				t.Fatalf("removal error = %v command:%v", model.err, cmd)
			}
		})
	}
}

func TestModelPullCleanEOFCompletesAndRefreshes(t *testing.T) {
	model, client := testModel(t)
	model.screen = ui.ScreenImagePull
	model.imagePullInput.SetValue("quay.io/example/empty-stream:latest")
	updated, cmd := model.Update(keyPress("", tea.KeyEnter))
	model = updated.(*Model)
	for steps := 0; cmd != nil && steps < 6; steps++ {
		updated, cmd = model.Update(cmd())
		model = updated.(*Model)
	}
	if cmd != nil || model.imagePullStatus != domain.ImageOperationSucceeded || !containsCall(client.Calls, string(domain.ActionImageList)) {
		t.Fatalf("clean EOF outcome = status:%q calls:%v command:%v", model.imagePullStatus, client.Calls, cmd)
	}
}

func TestModelImageRemovalNotFoundKeepsTargetSafe(t *testing.T) {
	model, client := testModel(t)
	image := testImage()
	client.Images = []domain.ImageSummary{image}
	model.images = client.Images
	model.screen = ui.ScreenImages
	updated, cmd := model.Update(keyPress("D", 'D'))
	model = updated.(*Model)
	client.Images = nil
	updated, cmd = model.Update(keyPress("y", 'y'))
	model = updated.(*Model)
	updated, cmd = model.Update(cmd())
	model = updated.(*Model)
	if cmd == nil {
		t.Fatal("not-found removal did not request an authoritative refresh")
	}
	updated, cmd = model.Update(cmd())
	model = updated.(*Model)
	if cmd != nil || model.err == nil || !strings.Contains(model.err.Error(), "Cible obsolète") {
		t.Fatalf("not-found removal = error:%v command:%v", model.err, cmd)
	}
}

func containsCall(calls []string, want string) bool {
	for _, call := range calls {
		if call == want {
			return true
		}
	}
	return false
}
