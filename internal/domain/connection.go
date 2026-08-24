package domain

import (
	"fmt"
	"net/url"
	"strings"
)

// ConnectionProfile is the non-secret metadata needed to reach a Podman service.
type ConnectionProfile struct {
	Name         string `json:"name"`
	URI          string `json:"uri"`
	IdentityPath string `json:"identity_path,omitempty"`
	Default      bool   `json:"default,omitempty"`
}

func (p ConnectionProfile) Validate() error {
	name := strings.TrimSpace(p.Name)
	if name == "" {
		return fmt.Errorf("connection name cannot be empty")
	}
	if len(name) > 64 {
		return fmt.Errorf("connection name cannot exceed 64 characters")
	}

	u, err := url.Parse(strings.TrimSpace(p.URI))
	if err != nil {
		return fmt.Errorf("invalid connection URI: %w", err)
	}
	if u.Scheme != "unix" && u.Scheme != "ssh" && u.Scheme != "tcp" {
		return fmt.Errorf("unsupported connection URI scheme %q", u.Scheme)
	}
	if u.Scheme == "ssh" && u.Host == "" {
		return fmt.Errorf("ssh connection URI must include a host")
	}
	if u.Scheme == "tcp" && u.Host == "" {
		return fmt.Errorf("tcp connection URI must include a host")
	}
	if u.Scheme == "unix" && u.Path == "" {
		return fmt.Errorf("unix connection URI must include a socket path")
	}
	return nil
}

func (p ConnectionProfile) DisplayName() string {
	if name := strings.TrimSpace(p.Name); name != "" {
		return name
	}
	return "Unnamed connection"
}
