package domain

import (
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
