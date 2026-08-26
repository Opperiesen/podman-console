package domain

import "fmt"

type Action string

const (
	ActionList            Action = "list"
	ActionInspect         Action = "inspect"
	ActionStart           Action = "start"
	ActionStop            Action = "stop"
	ActionRestart         Action = "restart"
	ActionRemove          Action = "remove"
	ActionContainerCreate Action = "container_create"
	ActionLogs            Action = "logs"
	ActionStats           Action = "stats"
	ActionImageList       Action = "image_list"
	ActionImageInspect    Action = "image_inspect"
	ActionImagePull       Action = "image_pull"
	ActionImageRemove     Action = "image_remove"
)

func (a Action) Destructive() bool {
	return a == ActionStop || a == ActionRestart || a == ActionRemove || a == ActionImageRemove
}

func (a Action) String() string { return string(a) }

type ErrorCategory string

const (
	ErrorInvalidConfig   ErrorCategory = "invalid_config"
	ErrorAuthorization   ErrorCategory = "authorization"
	ErrorTransport       ErrorCategory = "transport"
	ErrorHost            ErrorCategory = "host"
	ErrorStaleTarget     ErrorCategory = "stale_target"
	ErrorRegistry        ErrorCategory = "registry"
	ErrorInUse           ErrorCategory = "in_use"
	ErrorNameConflict    ErrorCategory = "name_conflict"
	ErrorPartial         ErrorCategory = "partial"
	ErrorMalformedStream ErrorCategory = "malformed_stream"
	ErrorCancelled       ErrorCategory = "cancelled"
	ErrorUnknown         ErrorCategory = "unknown"
)

type OperationError struct {
	Category ErrorCategory
	Action   Action
	TargetID string
	Err      error
}

func (e *OperationError) Error() string {
	if e == nil {
		return "unknown operation error"
	}
	if e.Err == nil {
		return fmt.Sprintf("%s operation failed", e.Action)
	}
	return e.Err.Error()
}

func (e *OperationError) Unwrap() error { return e.Err }
