package podman

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/Opperiesen/podman-console/internal/domain"
)

func TestWrapClassifiesErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		category domain.ErrorCategory
	}{
		{name: "cancelled", err: context.Canceled, category: domain.ErrorCancelled},
		{name: "deadline", err: context.DeadlineExceeded, category: domain.ErrorCancelled},
		{name: "network error", err: &net.DNSError{Err: "lookup failed", Name: "podman.example.test"}, category: domain.ErrorTransport},
		{name: "authorization", err: errors.New("remote returned permission denied"), category: domain.ErrorAuthorization},
		{name: "stale target", err: errors.New("no such container: abc"), category: domain.ErrorStaleTarget},
		{name: "transport marker", err: errors.New("connection refused"), category: domain.ErrorTransport},
		{name: "host", err: errors.New("podman service returned an unexpected response"), category: domain.ErrorHost},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Wrap(domain.ActionInspect, "container-id", tt.err)
			if err == nil {
				t.Fatal("Wrap() returned nil")
			}
			opErr, ok := err.(*domain.OperationError)
			if !ok {
				t.Fatalf("Wrap() type = %T, want *domain.OperationError", err)
			}
			if opErr.Category != tt.category || opErr.Action != domain.ActionInspect || opErr.TargetID != "container-id" {
				t.Fatalf("OperationError = %#v, want category %q/action inspect/target container-id", opErr, tt.category)
			}
			if !errors.Is(err, tt.err) {
				t.Fatal("Wrap() did not preserve the original error")
			}
			if got := ErrorMessage(err); got != tt.err.Error() {
				t.Fatalf("ErrorMessage() = %q, want %q", got, tt.err.Error())
			}
		})
	}
}

func TestWrapAndErrorMessageNilAndWrappedErrors(t *testing.T) {
	t.Parallel()

	if Wrap(domain.ActionList, "", nil) != nil {
		t.Fatal("Wrap(nil) did not return nil")
	}
	if got := ErrorMessage(nil); got != "" {
		t.Fatalf("ErrorMessage(nil) = %q", got)
	}

	base := errors.New("connection reset by peer")
	wrapped := fmt.Errorf("request failed: %w", base)
	err := Wrap(domain.ActionList, "", wrapped)
	if got := ErrorMessage(err); got != wrapped.Error() {
		t.Fatalf("ErrorMessage() = %q, want %q", got, wrapped.Error())
	}
}

func TestWrapClassifiesImageSpecificErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		message  string
		category domain.ErrorCategory
	}{
		{message: "manifest unknown: requested image", category: domain.ErrorRegistry},
		{message: "image is in use by a container", category: domain.ErrorInUse},
		{message: "failed to decode message from stream", category: domain.ErrorMalformedStream},
		{message: "no such image: abc", category: domain.ErrorStaleTarget},
	}
	for _, test := range tests {
		t.Run(test.message, func(t *testing.T) {
			err := Wrap(domain.ActionImagePull, "image", errors.New(test.message))
			operation, ok := err.(*domain.OperationError)
			if !ok || operation.Category != test.category {
				t.Fatalf("Wrap(%q) = %#v, want category %q", test.message, err, test.category)
			}
		})
	}
}
