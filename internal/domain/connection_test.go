package domain

import (
	"strings"
	"testing"
)

func TestConnectionProfileValidate(t *testing.T) {
	t.Parallel()

	valid := ConnectionProfile{Name: "local", URI: "unix:///run/user/1000/podman/podman.sock"}
	tests := []struct {
		name    string
		profile ConnectionProfile
		wantErr bool
	}{
		{name: "valid unix", profile: valid},
		{name: "valid ssh", profile: ConnectionProfile{Name: "remote", URI: "ssh://user@example.test/run/podman.sock"}},
		{name: "valid tcp", profile: ConnectionProfile{Name: "remote", URI: "tcp://example.test:8080"}},
		{name: "missing name", profile: ConnectionProfile{URI: valid.URI}, wantErr: true},
		{name: "name is whitespace", profile: ConnectionProfile{Name: "  ", URI: valid.URI}, wantErr: true},
		{name: "name too long", profile: ConnectionProfile{Name: strings.Repeat("x", 65), URI: valid.URI}, wantErr: true},
		{name: "unsupported scheme", profile: ConnectionProfile{Name: "local", URI: "http://example.test"}, wantErr: true},
		{name: "ssh without host", profile: ConnectionProfile{Name: "remote", URI: "ssh:///run/podman.sock"}, wantErr: true},
		{name: "tcp without host", profile: ConnectionProfile{Name: "remote", URI: "tcp:///podman"}, wantErr: true},
		{name: "unix without socket path", profile: ConnectionProfile{Name: "local", URI: "unix://"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.profile.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConnectionProfileDisplayName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want string
	}{
		{name: "  workstation  ", want: "workstation"},
		{name: "   ", want: "Unnamed connection"},
		{name: "", want: "Unnamed connection"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := ConnectionProfile{Name: tt.name}
			if got := profile.DisplayName(); got != tt.want {
				t.Fatalf("DisplayName() = %q, want %q", got, tt.want)
			}
		})
	}
}
