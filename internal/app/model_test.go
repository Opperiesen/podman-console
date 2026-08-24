package app

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/Opperiesen/podman-console/internal/config"
	"github.com/Opperiesen/podman-console/internal/domain"
	"github.com/Opperiesen/podman-console/internal/ui"
	"github.com/Opperiesen/podman-console/tests/fixtures"
)

func testProfile() domain.ConnectionProfile {
	return domain.ConnectionProfile{Name: "local", URI: "unix:///run/user/1000/podman/podman.sock"}
}

func testContainer() domain.ContainerSummary {
	return domain.ContainerSummary{ID: "abcdef0123456789", Name: "web", Image: "nginx:latest", State: domain.StateRunning}
}

func testModel(t *testing.T) (*Model, *fixtures.Client) {
	t.Helper()
	profile := testProfile()
	client := &fixtures.Client{Details: map[string]domain.ContainerDetails{}, Errors: map[domain.Action]error{}}
	client.Containers = []domain.ContainerSummary{testContainer()}
	client.Details[client.Containers[0].ID] = domain.ContainerDetails{ContainerSummary: client.Containers[0]}
	factory := &fixtures.Factory{Clients: map[string]*fixtures.Client{profile.Name: client}, Errors: map[string]error{}}
	store := config.NewStoreAt(filepath.Join(t.TempDir(), "config.json"))
	model := New(store, factory)
	model.file = config.File{Version: 1, Active: profile.Name, Profiles: []domain.ConnectionProfile{profile}}
	model.profile = profile
	model.client = client
	model.connected = true
	model.containers = client.Containers
	return &model, client
}

func keyPress(text string, code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Text: text, Code: code})
}

func TestModelLoadsActiveProfileAndInventory(t *testing.T) {
	profile := testProfile()
	container := testContainer()
	client := &fixtures.Client{Containers: []domain.ContainerSummary{container}, Details: map[string]domain.ContainerDetails{container.ID: {ContainerSummary: container}}, Errors: map[domain.Action]error{}}
	factory := &fixtures.Factory{Clients: map[string]*fixtures.Client{profile.Name: client}, Errors: map[string]error{}}
	file := config.File{Version: 1, Active: profile.Name, Profiles: []domain.ConnectionProfile{profile}}
	model := New(config.NewStoreAt(filepath.Join(t.TempDir(), "config.json")), factory)

	updated, cmd := model.Update(ConfigLoadedMsg{File: file})
	model = *(updated.(*Model))
	if cmd == nil {
		t.Fatal("ConfigLoadedMsg did not start a connection command")
	}
	updated, cmd = model.Update(cmd())
	model = *(updated.(*Model))
	if cmd != nil {
		t.Fatal("profile connection unexpectedly returned a follow-up command")
	}
	if !model.connected || model.profile.Name != profile.Name {
		t.Fatalf("model connection state = connected:%v profile:%q", model.connected, model.profile.Name)
	}
	if len(model.containers) != 1 || model.containers[0].Name != container.Name {
		t.Fatalf("model inventory = %#v", model.containers)
	}
}

func TestModelIgnoresStaleInventoryResponse(t *testing.T) {
	model, _ := testModel(t)
	model.generation = 4
	model.containers = []domain.ContainerSummary{testContainer()}
	updated, cmd := model.Update(InventoryLoadedMsg{Generation: 3, Containers: nil})
	model = updated.(*Model)
	if cmd != nil {
		t.Fatal("stale inventory response returned a command")
	}
	if len(model.containers) != 1 || model.containers[0].Name != "web" {
		t.Fatalf("stale response replaced inventory: %#v", model.containers)
	}
}

func TestDestructiveActionRequiresExactTargetAndCancellationDoesNotMutate(t *testing.T) {
	model, client := testModel(t)
	updated, cmd := model.Update(keyPress("x", 'x'))
	model = updated.(*Model)
	if cmd != nil {
		t.Fatal("stop request should wait for confirmation")
	}
	if model.mode != ui.ModeConfirm || model.pendingID != client.Containers[0].ID || model.pendingTarget != "web" {
		t.Fatalf("confirmation state = mode:%q target:%q id:%q", model.mode, model.pendingTarget, model.pendingID)
	}
	if len(client.Calls) != 0 {
		t.Fatalf("client mutated before confirmation: %v", client.Calls)
	}

	updated, cmd = model.Update(keyPress("", tea.KeyEscape))
	model = updated.(*Model)
	if cmd != nil || model.mode != ui.ModeNormal {
		t.Fatalf("cancel state = mode:%q command:%v", model.mode, cmd)
	}
	if len(client.Calls) != 0 || client.Containers[0].State != domain.StateRunning {
		t.Fatalf("cancelled stop changed client: calls=%v state=%s", client.Calls, client.Containers[0].State)
	}
}

func TestFilterPreservesOnlyMatchingContainers(t *testing.T) {
	model, _ := testModel(t)
	model.containers = append(model.containers, domain.ContainerSummary{ID: "second", Name: "db", Image: "postgres:latest", State: domain.StateExited})
	model.filter = "POSTGRES"
	visible := model.visibleContainers()
	if len(visible) != 1 || visible[0].Name != "db" {
		t.Fatalf("visible containers = %#v", visible)
	}
}

func TestStreamCancellationKeepsPartialLogs(t *testing.T) {
	model, _ := testModel(t)
	model.screen = ui.ScreenLogs
	model.streamReturn = ui.ScreenInventory
	line := domain.LogLine{Text: "started", Stream: "stdout"}
	updated, cmd := model.Update(logStreamEvent{Line: &line, Next: make(chan logStreamEvent)})
	model = updated.(*Model)
	if cmd == nil || len(model.logLines) != 1 || model.logLines[0].Text != "started" {
		t.Fatalf("partial log state = lines:%#v command:%v", model.logLines, cmd)
	}
	updated, cmd = model.Update(logStreamEvent{Done: true, Err: context.Canceled})
	model = updated.(*Model)
	if cmd != nil || !model.streamStopped || model.err != nil {
		t.Fatalf("cancelled stream state = stopped:%v err:%v cmd:%v", model.streamStopped, model.err, cmd)
	}
}

func TestFriendlyErrorPreservesActionableCategories(t *testing.T) {
	err := &domain.OperationError{Category: domain.ErrorAuthorization, Action: domain.ActionStart, Err: errors.New("permission denied")}
	if got := friendlyError(err).Error(); got != "Autorisation refusée : permission denied" {
		t.Fatalf("friendlyError() = %q", got)
	}
}
