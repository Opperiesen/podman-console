package domain

import "testing"

func TestNormalizeContainerState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  ContainerState
	}{
		{input: "created", want: StateCreated},
		{input: " CONFIGURED ", want: StateCreated},
		{input: "Running", want: StateRunning},
		{input: "paused", want: StatePaused},
		{input: "stopped", want: StateStopped},
		{input: "dead", want: StateExited},
		{input: " EXITED ", want: StateExited},
		{input: "restarting", want: StateUnknown},
		{input: "", want: StateUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := NormalizeContainerState(tt.input); got != tt.want {
				t.Fatalf("NormalizeContainerState(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestContainerStateString(t *testing.T) {
	if got := StateRunning.String(); got != "running" {
		t.Fatalf("StateRunning.String() = %q, want %q", got, "running")
	}
}
