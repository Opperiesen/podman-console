package ui

import (
	"fmt"
	"strings"
	"testing"

	"charm.land/bubbles/v2/help"
)

func TestNarrowLogViewportKeepsNewestLinesAndStoppedState(t *testing.T) {
	var lines []string
	for i := 0; i < 20; i++ {
		lines = append(lines, fmt.Sprintf("line-%02d", i))
	}
	output := Render(ViewData{Width: 34, Height: 12, Screen: ScreenLogs, Mode: ModeNormal, LogContent: strings.Join(lines, "\n"), StreamStopped: true, Keys: NewKeyMap(), Help: help.New()})
	if !strings.Contains(output, "line-19") || !strings.Contains(output, "flux arrêté") {
		t.Fatalf("narrow/stopped logs lost newest data: %s", output)
	}
	if strings.Contains(output, "line-00") {
		t.Fatalf("narrow log viewport did not enforce overflow: %s", output)
	}
}
