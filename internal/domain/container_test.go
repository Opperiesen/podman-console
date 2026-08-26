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

func TestContainerCreateRequestValidation(t *testing.T) {
	t.Parallel()

	valid := ContainerCreateRequest{ImageID: "sha256:image", Name: "web_1", Command: []string{"sleep", "60"}}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}
	for _, name := range []string{"", " web", "web name", "-web", "web/name"} {
		request := valid
		request.Name = name
		if err := request.Validate(); err == nil {
			t.Errorf("request name %q was accepted", name)
		}
	}
	invalidArgument := valid
	invalidArgument.Command = []string{"sleep\x00"}
	if err := invalidArgument.Validate(); err == nil {
		t.Error("NUL command argument was accepted")
	}
}

func TestParseContainerCommand(t *testing.T) {
	t.Parallel()

	if got, err := ParseContainerCommand("  sleep   60 "); err != nil || len(got) != 2 || got[0] != "sleep" || got[1] != "60" {
		t.Fatalf("ParseContainerCommand() = %#v, %v", got, err)
	}
	if got, err := ParseContainerCommand("   "); err != nil || got != nil {
		t.Fatalf("blank ParseContainerCommand() = %#v, %v", got, err)
	}
	for _, command := range []string{"sh -c 'sleep 60'", "echo ok | cat", "echo $HOME", "echo\\ value"} {
		if _, err := ParseContainerCommand(command); err == nil {
			t.Errorf("unsupported shell command %q was accepted", command)
		}
	}
}
