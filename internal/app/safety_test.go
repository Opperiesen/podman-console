package app

import (
	"testing"

	"github.com/Opperiesen/podman-console/internal/domain"
	"github.com/Opperiesen/podman-console/internal/ui"
)

func TestEveryDestructiveBindingRequiresConfirmation(t *testing.T) {
	bindings := []struct {
		name string
		key  string
		code rune
	}{
		{name: "stop", key: "x", code: 'x'},
		{name: "restart", key: "R", code: 'R'},
		{name: "remove", key: "D", code: 'D'},
	}
	for _, test := range bindings {
		t.Run(test.name, func(t *testing.T) {
			model, client := testModel(t)
			updated, cmd := model.Update(keyPress(test.key, test.code))
			model = updated.(*Model)
			if cmd != nil || model.mode != ui.ModeConfirm || model.pendingAction.Destructive() == false {
				t.Fatalf("confirmation state = mode:%q action:%q command:%v", model.mode, model.pendingAction, cmd)
			}
			if len(client.Calls) != 0 {
				t.Fatalf("mutation sent before confirmation: %v", client.Calls)
			}
			updated, cmd = model.Update(keyPress("", 'n'))
			model = updated.(*Model)
			if cmd != nil || model.mode != ui.ModeNormal || len(client.Calls) != 0 {
				t.Fatalf("cancel did not invalidate mutation: mode:%q calls:%v command:%v", model.mode, client.Calls, cmd)
			}
		})
	}
}

func TestConfirmedOperationUsesExactSelectedID(t *testing.T) {
	model, client := testModel(t)
	updated, cmd := model.Update(keyPress("x", 'x'))
	model = updated.(*Model)
	updated, cmd = model.Update(keyPress("", 'y'))
	model = updated.(*Model)
	if cmd == nil {
		t.Fatal("confirmed stop did not create an operation command")
	}
	updated, cmd = model.Update(cmd())
	model = updated.(*Model)
	if cmd == nil {
		t.Fatal("successful operation did not schedule authoritative refresh")
	}
	if len(client.Calls) == 0 || client.Calls[0] != string(domain.ActionStop)+":"+client.Containers[0].ID {
		t.Fatalf("operation target = %v", client.Calls)
	}
}
