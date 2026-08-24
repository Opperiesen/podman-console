package config

import (
	"strings"
	"testing"

	"github.com/Opperiesen/podman-console/internal/domain"
)

func profile(name, uri string) domain.ConnectionProfile {
	return domain.ConnectionProfile{Name: name, URI: uri}
}

func TestDefault(t *testing.T) {
	got := Default()
	if got.Version != currentVersion {
		t.Fatalf("Default().Version = %d, want %d", got.Version, currentVersion)
	}
	if got.Profiles == nil {
		t.Fatal("Default().Profiles is nil")
	}
	if len(got.Profiles) != 0 {
		t.Fatalf("Default().Profiles has %d entries, want 0", len(got.Profiles))
	}
}

func TestFileValidate(t *testing.T) {
	t.Parallel()

	validProfile := profile("local", "unix:///run/podman.sock")
	tests := []struct {
		name string
		file File
	}{
		{name: "missing version", file: File{Profiles: []domain.ConnectionProfile{validProfile}}},
		{name: "unsupported version", file: File{Version: currentVersion + 1}},
		{name: "duplicate names ignoring case and spaces", file: File{Version: currentVersion, Profiles: []domain.ConnectionProfile{
			profile(" Local ", validProfile.URI), profile("local", validProfile.URI),
		}}},
		{name: "multiple defaults", file: File{Version: currentVersion, Profiles: []domain.ConnectionProfile{
			{Name: "one", URI: validProfile.URI, Default: true}, {Name: "two", URI: validProfile.URI, Default: true},
		}}},
		{name: "active profile missing", file: File{Version: currentVersion, Active: "missing", Profiles: []domain.ConnectionProfile{validProfile}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.file.Validate(); err == nil {
				t.Fatal("Validate() returned nil, want error")
			}
		})
	}

	if err := (File{Version: currentVersion, Profiles: []domain.ConnectionProfile{validProfile}, Active: " LOCAL "}).Validate(); err != nil {
		t.Fatalf("valid file rejected: %v", err)
	}
}

func TestFileProfileAndActiveProfile(t *testing.T) {
	t.Parallel()

	local := profile("local", "unix:///run/podman.sock")
	remote := profile("remote", "ssh://user@example.test/run/podman.sock")

	file := File{Version: currentVersion, Profiles: []domain.ConnectionProfile{local, remote}, Active: " LOCAL "}
	got, ok := file.Profile(" local ")
	if !ok || got.Name != local.Name {
		t.Fatalf("Profile() = %#v, %v; want local profile", got, ok)
	}
	if _, ok := file.Profile("missing"); ok {
		t.Fatal("Profile(missing) found a profile")
	}
	got, ok = file.ActiveProfile()
	if !ok || got.Name != local.Name {
		t.Fatalf("ActiveProfile() = %#v, %v; want local profile", got, ok)
	}

	defaultFile := File{Version: currentVersion, Profiles: []domain.ConnectionProfile{local, {Name: remote.Name, URI: remote.URI, Default: true}}}
	got, ok = defaultFile.ActiveProfile()
	if !ok || got.Name != remote.Name {
		t.Fatalf("default ActiveProfile() = %#v, %v; want remote profile", got, ok)
	}

	single := File{Version: currentVersion, Profiles: []domain.ConnectionProfile{remote}}
	got, ok = single.ActiveProfile()
	if !ok || got.Name != remote.Name {
		t.Fatalf("single ActiveProfile() = %#v, %v; want remote profile", got, ok)
	}
	if _, ok := (File{Version: currentVersion}).ActiveProfile(); ok {
		t.Fatal("empty ActiveProfile() found a profile")
	}
}

func TestFileUpsert(t *testing.T) {
	t.Parallel()

	file := Default()
	local := profile("local", "unix:///run/podman.sock")
	if err := file.Upsert(local); err != nil {
		t.Fatalf("first Upsert() error = %v", err)
	}
	if len(file.Profiles) != 1 || !file.Profiles[0].Default || file.Active != "local" {
		t.Fatalf("first Upsert() state = %#v, active %q", file.Profiles, file.Active)
	}

	updated := profile("LOCAL", "ssh://user@example.test/run/podman.sock")
	if err := file.Upsert(updated); err != nil {
		t.Fatalf("update Upsert() error = %v", err)
	}
	if len(file.Profiles) != 1 || file.Profiles[0].URI != updated.URI || file.Active != "local" {
		t.Fatalf("update Upsert() state = %#v, active %q", file.Profiles, file.Active)
	}

	if err := file.Upsert(profile("bad", "http://example.test")); err == nil {
		t.Fatal("invalid Upsert() returned nil")
	}
}

func TestFileRemove(t *testing.T) {
	t.Parallel()

	file := File{Version: currentVersion, Active: "two", Profiles: []domain.ConnectionProfile{
		profile("one", "unix:///one.sock"), profile("two", "unix:///two.sock"),
	}}
	if !file.Remove("TWO") {
		t.Fatal("Remove() returned false for existing profile")
	}
	if len(file.Profiles) != 1 || file.Profiles[0].Name != "one" || file.Active != "one" {
		t.Fatalf("Remove() state = %#v, active %q", file.Profiles, file.Active)
	}
	if file.Remove("missing") {
		t.Fatal("Remove() returned true for missing profile")
	}

	if strings.TrimSpace(file.Profiles[0].Name) == "" {
		t.Fatal("remaining profile unexpectedly empty")
	}
}
