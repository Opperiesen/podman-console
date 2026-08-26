package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

type ContainerState string

const (
	StateCreated ContainerState = "created"
	StateRunning ContainerState = "running"
	StatePaused  ContainerState = "paused"
	StateStopped ContainerState = "stopped"
	StateExited  ContainerState = "exited"
	StateUnknown ContainerState = "unknown"
)

func NormalizeContainerState(value string) ContainerState {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "created", "configured":
		return StateCreated
	case "running":
		return StateRunning
	case "paused":
		return StatePaused
	case "stopped":
		return StateStopped
	case "exited", "dead":
		return StateExited
	default:
		return StateUnknown
	}
}

func (s ContainerState) String() string { return string(s) }

type PortMapping struct {
	HostIP        string
	HostPort      uint16
	ContainerPort uint16
	Protocol      string
}

type Mount struct {
	Type        string
	Source      string
	Destination string
	Mode        string
	ReadWrite   bool
}

type NetworkAttachment struct {
	Name       string
	IPAddress  string
	MACAddress string
}

type ContainerSummary struct {
	ID        string
	Name      string
	Image     string
	State     ContainerState
	Status    string
	Ports     []PortMapping
	CreatedAt time.Time
}

type ContainerDetails struct {
	ContainerSummary
	Command    []string
	Entrypoint []string
	Mounts     []Mount
	Networks   []NetworkAttachment
	Labels     map[string]string
	WorkingDir string
}

type ContainerCreateRequest struct {
	ImageID        string
	ImageReference string
	Name           string
	Command        []string
}

func (r ContainerCreateRequest) Validate() error {
	if strings.TrimSpace(r.ImageID) == "" {
		return fmt.Errorf("image ID cannot be empty")
	}
	if !containerNamePattern.MatchString(r.Name) {
		return fmt.Errorf("container name must be 1-63 characters and contain only letters, numbers, '.', '_' or '-'")
	}
	for _, argument := range r.Command {
		if argument == "" || strings.ContainsRune(argument, '\x00') {
			return fmt.Errorf("container command contains an empty or NUL argument")
		}
	}
	return nil
}

var containerNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,62}$`)

func ParseContainerCommand(value string) ([]string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	if strings.ContainsAny(value, "|&;<>()$'\"\\`") || strings.ContainsRune(value, '\x00') {
		return nil, fmt.Errorf("command must contain arguments only; shell operators and quoting are not supported")
	}
	return strings.Fields(value), nil
}

type ContainerRunResult struct {
	ContainerID string
	Started     bool
	Warnings    []string
}

type ContainerCreateStatus string

const (
	ContainerCreateIdle       ContainerCreateStatus = "idle"
	ContainerCreateEditing    ContainerCreateStatus = "editing"
	ContainerCreateConfirming ContainerCreateStatus = "confirming"
	ContainerCreateCreating   ContainerCreateStatus = "creating"
	ContainerCreateStarting   ContainerCreateStatus = "starting"
	ContainerCreateRefreshing ContainerCreateStatus = "refreshing"
	ContainerCreateSucceeded  ContainerCreateStatus = "succeeded"
	ContainerCreatePartial    ContainerCreateStatus = "partial"
	ContainerCreateFailed     ContainerCreateStatus = "failed"
	ContainerCreateCancelled  ContainerCreateStatus = "cancelled"
)

type ContainerStats struct {
	ContainerID      string
	CPUPercent       float64
	MemoryUsageBytes uint64
	MemoryLimitBytes uint64
	MemoryPercent    float64
	ObservedAt       time.Time
}

type LogLine struct {
	Text       string
	Stream     string
	ObservedAt time.Time
}
