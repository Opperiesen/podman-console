//go:build integration

package integration

import (
	"context"
	"os"
	"testing"

	"github.com/Opperiesen/podman-console/internal/domain"
	"github.com/Opperiesen/podman-console/internal/podman"
)

func TestLiveInventory(t *testing.T) {
	uri := os.Getenv("PODMAN_CONSOLE_URI")
	if uri == "" {
		t.Skip("PODMAN_CONSOLE_URI is not set")
	}
	profile := domain.ConnectionProfile{Name: "integration", URI: uri, IdentityPath: os.Getenv("PODMAN_CONSOLE_IDENTITY")}
	client, err := (podman.BindingsFactory{}).Connect(context.Background(), profile)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if _, err := client.ListContainers(context.Background()); err != nil {
		t.Fatalf("list containers: %v", err)
	}
}
