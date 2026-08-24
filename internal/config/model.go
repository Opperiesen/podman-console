package config

import (
	"fmt"
	"strings"

	"github.com/Opperiesen/podman-console/internal/domain"
)

const currentVersion = 1

type File struct {
	Version  int                        `json:"version"`
	Profiles []domain.ConnectionProfile `json:"profiles"`
	Active   string                     `json:"active,omitempty"`
}

func Default() File {
	return File{Version: currentVersion, Profiles: []domain.ConnectionProfile{}}
}

func (f File) Validate() error {
	if f.Version == 0 {
		return fmt.Errorf("missing config version")
	}
	if f.Version != currentVersion {
		return fmt.Errorf("unsupported config version %d", f.Version)
	}

	seen := make(map[string]struct{}, len(f.Profiles))
	defaults := 0
	for _, profile := range f.Profiles {
		if err := profile.Validate(); err != nil {
			return fmt.Errorf("profile %q: %w", profile.Name, err)
		}
		key := strings.ToLower(strings.TrimSpace(profile.Name))
		if _, ok := seen[key]; ok {
			return fmt.Errorf("duplicate connection profile %q", profile.Name)
		}
		seen[key] = struct{}{}
		if profile.Default {
			defaults++
		}
	}
	if defaults > 1 {
		return fmt.Errorf("only one connection profile can be default")
	}
	if f.Active != "" {
		if _, ok := f.Profile(f.Active); !ok {
			return fmt.Errorf("active connection profile %q does not exist", f.Active)
		}
	}
	return nil
}

func (f File) Profile(name string) (domain.ConnectionProfile, bool) {
	for _, profile := range f.Profiles {
		if strings.EqualFold(strings.TrimSpace(profile.Name), strings.TrimSpace(name)) {
			return profile, true
		}
	}
	return domain.ConnectionProfile{}, false
}

func (f File) ActiveProfile() (domain.ConnectionProfile, bool) {
	if f.Active != "" {
		return f.Profile(f.Active)
	}
	for _, profile := range f.Profiles {
		if profile.Default {
			return profile, true
		}
	}
	if len(f.Profiles) == 1 {
		return f.Profiles[0], true
	}
	return domain.ConnectionProfile{}, false
}

func (f *File) Upsert(profile domain.ConnectionProfile) error {
	if err := profile.Validate(); err != nil {
		return err
	}
	for i := range f.Profiles {
		if strings.EqualFold(f.Profiles[i].Name, profile.Name) {
			f.Profiles[i] = profile
			if f.Active == "" {
				f.Active = profile.Name
			}
			return f.Validate()
		}
	}
	if len(f.Profiles) == 0 {
		profile.Default = true
	}
	f.Profiles = append(f.Profiles, profile)
	if f.Active == "" {
		f.Active = profile.Name
	}
	return f.Validate()
}

func (f *File) Remove(name string) bool {
	for i, profile := range f.Profiles {
		if strings.EqualFold(profile.Name, name) {
			f.Profiles = append(f.Profiles[:i], f.Profiles[i+1:]...)
			if strings.EqualFold(f.Active, name) {
				f.Active = ""
				if len(f.Profiles) > 0 {
					f.Active = f.Profiles[0].Name
				}
			}
			return true
		}
	}
	return false
}
