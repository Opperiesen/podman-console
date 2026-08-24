package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Opperiesen/podman-console/internal/domain"
)

func TestStoreLoadMissingReturnsDefault(t *testing.T) {
	store := NewStoreAt(filepath.Join(t.TempDir(), "nested", "config.json"))

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.Version != currentVersion || len(got.Profiles) != 0 || got.Active != "" {
		t.Fatalf("Load() = %#v, want default config", got)
	}
}

func TestStoreSaveLoadRoundTripAndPermissions(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "config", "config.json")
	store := NewStoreAt(path)
	file := File{Version: currentVersion, Active: "local", Profiles: []domain.ConnectionProfile{
		{Name: "local", URI: "unix:///run/podman.sock", IdentityPath: "/home/user/.ssh/id_ed25519"},
	}}

	if err := store.Save(file); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load() after Save() error = %v", err)
	}
	if got.Version != file.Version || got.Active != file.Active || len(got.Profiles) != 1 || got.Profiles[0] != file.Profiles[0] {
		t.Fatalf("round trip = %#v, want %#v", got, file)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(config) error = %v", err)
	}
	if gotMode := info.Mode().Perm(); gotMode != 0o600 {
		t.Fatalf("config mode = %04o, want 0600", gotMode)
	}
	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("Stat(config directory) error = %v", err)
	}
	if gotMode := dirInfo.Mode().Perm(); gotMode != 0o700 {
		t.Fatalf("config directory mode = %04o, want 0700", gotMode)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(config) error = %v", err)
	}
	if len(data) == 0 || data[len(data)-1] != '\n' {
		t.Fatalf("config does not end with a newline: %q", data)
	}
	var decoded File
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("saved config is not valid JSON: %v", err)
	}
	tmpFiles, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".config-*.tmp"))
	if err != nil {
		t.Fatalf("Glob(temporary configs) error = %v", err)
	}
	if len(tmpFiles) != 0 {
		t.Fatalf("temporary config files remain after Save(): %v", tmpFiles)
	}
}

func TestStoreSaveRejectsInvalidConfigWithoutReplacingExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	store := NewStoreAt(path)
	valid := File{Version: currentVersion, Profiles: []domain.ConnectionProfile{{Name: "local", URI: "unix:///run/podman.sock"}}}
	if err := store.Save(valid); err != nil {
		t.Fatalf("initial Save() error = %v", err)
	}
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(original) error = %v", err)
	}

	invalid := File{Version: currentVersion, Profiles: []domain.ConnectionProfile{{Name: "local", URI: "http://example.test"}}}
	if err := store.Save(invalid); err == nil {
		t.Fatal("invalid Save() returned nil")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(after) error = %v", err)
	}
	if string(after) != string(original) {
		t.Fatal("invalid Save() replaced the existing config")
	}
}

func TestStoreLoadErrors(t *testing.T) {
	t.Parallel()

	t.Run("invalid json", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := NewStoreAt(path).Load()
		if err == nil {
			t.Fatal("Load() returned nil for invalid JSON")
		}
	})

	t.Run("invalid config", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.json")
		data, err := json.Marshal(File{Version: currentVersion + 1})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		_, err = NewStoreAt(path).Load()
		if err == nil {
			t.Fatal("Load() returned nil for invalid config")
		}
	})

	t.Run("read failure", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "directory")
		if err := os.Mkdir(path, 0o700); err != nil {
			t.Fatal(err)
		}
		_, err := NewStoreAt(path).Load()
		if err == nil || errors.Is(err, os.ErrNotExist) {
			t.Fatalf("Load() error = %v, want non-not-exist error", err)
		}
	})
}
