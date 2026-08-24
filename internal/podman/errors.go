package podman

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/Opperiesen/podman-console/internal/domain"
)

func Wrap(action domain.Action, target string, err error) error {
	if err == nil {
		return nil
	}
	category := classify(err)
	return &domain.OperationError{Category: category, Action: action, TargetID: target, Err: err}
}

func classify(err error) domain.ErrorCategory {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return domain.ErrorCancelled
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return domain.ErrorTransport
	}
	message := strings.ToLower(err.Error())
	for _, marker := range []string{"permission denied", "unauthorized", "forbidden", "authentication", "auth failed"} {
		if strings.Contains(message, marker) {
			return domain.ErrorAuthorization
		}
	}
	for _, marker := range []string{"no such container", "container not found", "does not exist", "not found"} {
		if strings.Contains(message, marker) {
			return domain.ErrorStaleTarget
		}
	}
	for _, marker := range []string{"connection refused", "connection reset", "no such file or directory", "unable to connect", "timeout"} {
		if strings.Contains(message, marker) {
			return domain.ErrorTransport
		}
	}
	return domain.ErrorHost
}

func ErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	var opErr *domain.OperationError
	if errors.As(err, &opErr) {
		if opErr.Err != nil {
			return opErr.Err.Error()
		}
	}
	return fmt.Sprint(err)
}
