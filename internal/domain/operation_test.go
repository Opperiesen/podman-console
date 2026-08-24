package domain

import (
	"errors"
	"testing"
)

func TestActionDestructive(t *testing.T) {
	t.Parallel()

	for _, action := range []Action{ActionList, ActionInspect, ActionStart, ActionLogs, ActionStats} {
		if action.Destructive() {
			t.Errorf("%q is destructive", action)
		}
	}
	for _, action := range []Action{ActionStop, ActionRestart, ActionRemove} {
		if !action.Destructive() {
			t.Errorf("%q is not destructive", action)
		}
	}
}

func TestOperationError(t *testing.T) {
	t.Parallel()

	underlying := errors.New("permission denied")
	err := &OperationError{Category: ErrorAuthorization, Action: ActionStop, TargetID: "abc123", Err: underlying}
	if got := err.Error(); got != underlying.Error() {
		t.Fatalf("Error() = %q, want %q", got, underlying)
	}
	if !errors.Is(err, underlying) {
		t.Fatal("OperationError does not unwrap its underlying error")
	}

	withoutCause := &OperationError{Action: ActionList}
	if got := withoutCause.Error(); got != "list operation failed" {
		t.Fatalf("Error() without cause = %q", got)
	}

	var nilErr *OperationError
	if got := nilErr.Error(); got != "unknown operation error" {
		t.Fatalf("nil Error() = %q", got)
	}
}
