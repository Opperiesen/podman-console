package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/Opperiesen/podman-console/internal/domain"
	"github.com/Opperiesen/podman-console/internal/ui"
)

func TestDetailLoadTranslatesSelectedContainerAndPreservesTarget(t *testing.T) {
	model, _ := testModel(t)
	updated, cmd := model.Update(keyPress("", tea.KeyEnter))
	model = updated.(*Model)
	if cmd == nil || model.screen != ui.ScreenDetails || model.detailLoading == false {
		t.Fatalf("detail state = screen:%q loading:%v command:%v", model.screen, model.detailLoading, cmd)
	}
	message := cmd()
	updated, cmd = model.Update(message)
	model = updated.(*Model)
	if cmd != nil || model.details == nil || model.details.ID != model.containers[0].ID {
		t.Fatalf("detail result = %#v command:%v", model.details, cmd)
	}
	if model.details.Name != "web" {
		t.Fatalf("detail name = %q, want web", model.details.Name)
	}
}

func TestStaleDetailResponseDoesNotReplaceCurrentTarget(t *testing.T) {
	model, _ := testModel(t)
	model.screen = ui.ScreenDetails
	model.generation = 8
	current := domain.ContainerDetails{ContainerSummary: domain.ContainerSummary{ID: "current", Name: "current"}}
	model.details = &current
	stale := domain.ContainerDetails{ContainerSummary: domain.ContainerSummary{ID: "stale", Name: "stale"}}
	updated, cmd := model.Update(DetailsLoadedMsg{Generation: 7, Details: stale})
	model = updated.(*Model)
	if cmd != nil || model.details.ID != "current" {
		t.Fatalf("stale detail replaced current target: %#v", model.details)
	}
}
