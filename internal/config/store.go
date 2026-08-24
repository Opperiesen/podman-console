package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type Store struct {
	path string
}

func NewStore(appName string) (*Store, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("resolve user config directory: %w", err)
	}
	return NewStoreAt(filepath.Join(dir, appName, "config.json")), nil
}

func NewStoreAt(path string) *Store { return &Store{path: path} }

func (s *Store) Path() string { return s.path }

func (s *Store) Load() (File, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return File{}, fmt.Errorf("read config %s: %w", s.path, err)
	}
	var file File
	if err := json.Unmarshal(data, &file); err != nil {
		return File{}, fmt.Errorf("parse config %s: %w", s.path, err)
	}
	if err := file.Validate(); err != nil {
		return File{}, fmt.Errorf("validate config %s: %w", s.path, err)
	}
	return file, nil
}

func (s *Store) Save(file File) error {
	if file.Version == 0 {
		file.Version = currentVersion
	}
	if err := file.Validate(); err != nil {
		return fmt.Errorf("validate config before save: %w", err)
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	if err := os.Chmod(filepath.Dir(s.path), 0o700); err != nil && runtime.GOOS != "windows" {
		return fmt.Errorf("protect config directory: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary config: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("protect temporary config: %w", err)
	}
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temporary config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary config: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		if runtime.GOOS != "windows" {
			return fmt.Errorf("replace config: %w", err)
		}
		// Windows does not replace an existing destination with Rename. The
		// temporary file is already complete and protected; remove the old
		// snapshot only after that write has succeeded, then move it in place.
		if removeErr := os.Remove(s.path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return fmt.Errorf("replace config: %w", err)
		}
		if renameErr := os.Rename(tmpName, s.path); renameErr != nil {
			return fmt.Errorf("replace config after removing old snapshot: %w", renameErr)
		}
	}
	return nil
}
