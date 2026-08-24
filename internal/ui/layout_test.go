package ui

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/help"
	"github.com/Opperiesen/podman-console/internal/domain"
)

func TestRenderInventoryIncludesTargetRowsAndState(t *testing.T) {
	output := Render(ViewData{
		Width:      120,
		Height:     30,
		Screen:     ScreenInventory,
		Mode:       ModeNormal,
		Profile:    domain.ConnectionProfile{Name: "local", URI: "unix:///run/podman.sock"},
		Connected:  true,
		Containers: []domain.ContainerSummary{{ID: "abcdef0123456789", Name: "web", Image: "nginx:latest", State: domain.StateRunning}},
		Keys:       NewKeyMap(),
		Help:       help.New(),
	})
	for _, want := range []string{"local", "nginx:latest", "running", "CIBLE", "PODMAN CONSOLE"} {
		if !strings.Contains(output, want) {
			t.Errorf("rendered inventory does not contain %q:\n%s", want, output)
		}
	}
}

func TestRenderEmptyAndConnectionErrorStates(t *testing.T) {
	output := Render(ViewData{
		Width:   80,
		Height:  24,
		Screen:  ScreenInventory,
		Mode:    ModeNormal,
		Profile: domain.ConnectionProfile{Name: "remote", URI: "ssh://user@example.test/run/podman.sock"},
		Error:   errTest("Cible injoignable : timeout"),
		Keys:    NewKeyMap(),
		Help:    help.New(),
	})
	for _, want := range []string{"Aucun conteneur", "Cible injoignable", "remote"} {
		if !strings.Contains(output, want) {
			t.Errorf("rendered empty/error state does not contain %q:\n%s", want, output)
		}
	}
}

func TestRenderConfirmationNamesExactTarget(t *testing.T) {
	output := Render(ViewData{
		Width:           100,
		Height:          30,
		Screen:          ScreenInventory,
		Mode:            ModeConfirm,
		Profile:         domain.ConnectionProfile{Name: "local", URI: "unix:///run/podman.sock"},
		ConfirmAction:   "Arrêt",
		ConfirmTarget:   "web-prod",
		ConfirmTargetID: "abcdef0123456789",
		Keys:            NewKeyMap(),
		Help:            help.New(),
	})
	if !strings.Contains(output, "web-prod") {
		t.Fatalf("confirmation target is not rendered exactly:\n%s", output)
	}
}

func TestRenderLogsAndStoppedStatsPreserveData(t *testing.T) {
	logs := Render(ViewData{Width: 80, Height: 24, Screen: ScreenLogs, Mode: ModeNormal, LogContent: "[stdout] first line", StreamStopped: true, Keys: NewKeyMap(), Help: help.New()})
	if !strings.Contains(logs, "first line") || !strings.Contains(logs, "flux arrêté") {
		t.Fatalf("logs rendering = %s", logs)
	}
	stats := Render(ViewData{Width: 80, Height: 24, Screen: ScreenStats, Mode: ModeNormal, StreamStopped: true, Stats: &domain.ContainerStats{CPUPercent: 3.5, MemoryUsageBytes: 1024, MemoryLimitBytes: 2048}, Keys: NewKeyMap(), Help: help.New()})
	if !strings.Contains(stats, "3.50%") || !strings.Contains(stats, "dernière mesure") {
		t.Fatalf("stats rendering = %s", stats)
	}
}

type errTest string

func (e errTest) Error() string { return string(e) }
