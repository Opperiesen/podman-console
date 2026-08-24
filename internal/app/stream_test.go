package app

import (
	"context"
	"errors"
	"testing"

	"github.com/Opperiesen/podman-console/internal/domain"
	"github.com/Opperiesen/podman-console/internal/podman"
	"github.com/Opperiesen/podman-console/internal/ui"
	"github.com/Opperiesen/podman-console/tests/fixtures"
)

func TestLogOrderingAndPartialErrorPreservation(t *testing.T) {
	model, _ := testModel(t)
	model.screen = ui.ScreenLogs
	model.streamGeneration = 4
	first := domain.LogLine{Text: "one", Stream: "stdout"}
	second := domain.LogLine{Text: "two", Stream: "stderr"}
	updated, cmd := model.Update(logStreamEvent{Generation: 4, Line: &first, Next: make(chan logStreamEvent)})
	model = updated.(*Model)
	if cmd == nil {
		t.Fatal("first stream event did not schedule the next read")
	}
	updated, cmd = model.Update(logStreamEvent{Generation: 4, Line: &second, Next: make(chan logStreamEvent)})
	model = updated.(*Model)
	if cmd == nil || len(model.logLines) != 2 || model.logLines[0].Text != "one" || model.logLines[1].Text != "two" {
		t.Fatalf("log order = %#v command:%v", model.logLines, cmd)
	}
	streamErr := errors.New("stream interrupted")
	updated, cmd = model.Update(logStreamEvent{Generation: 4, Done: true, Err: streamErr})
	model = updated.(*Model)
	if cmd != nil || !model.streamStopped || len(model.logLines) != 2 || model.err == nil {
		t.Fatalf("partial error state = lines:%d stopped:%v err:%v command:%v", len(model.logLines), model.streamStopped, model.err, cmd)
	}
}

func TestOldStreamEventsAreIgnoredAfterCancellation(t *testing.T) {
	model, _ := testModel(t)
	model.screen = ui.ScreenLogs
	model.streamGeneration = 9
	line := domain.LogLine{Text: "old"}
	updated, cmd := model.Update(logStreamEvent{Generation: 8, Line: &line, Done: true})
	model = updated.(*Model)
	if cmd != nil || len(model.logLines) != 0 || model.streamStopped {
		t.Fatalf("old stream event changed state: lines:%v stopped:%v", model.logLines, model.streamStopped)
	}
}

func TestStreamStartCommandsReturnEventsRatherThanNestedCommands(t *testing.T) {
	client := &fixtures.Client{
		Logs:  []domain.LogLine{{Text: "live log", Stream: "stdout"}},
		Stats: []domain.ContainerStats{{ContainerID: "container-id", CPUPercent: 1.5}},
	}

	logCtx, cancelLogs := context.WithCancel(context.Background())
	logMsg := startLogStreamCmd(logCtx, client, "container-id", podman.LogOptions{Follow: true}, 7)()
	logEvent, ok := logMsg.(logStreamEvent)
	if !ok || logEvent.Line == nil || logEvent.Line.Text != "live log" || logEvent.Generation != 7 {
		t.Fatalf("first log message = %#v, want generation 7 log event", logMsg)
	}
	cancelLogs()

	statsCtx, cancelStats := context.WithCancel(context.Background())
	statsMsg := startStatsStreamCmd(statsCtx, client, "container-id", 8)()
	statsEvent, ok := statsMsg.(statsStreamEvent)
	if !ok || statsEvent.Sample == nil || statsEvent.Sample.CPUPercent != 1.5 || statsEvent.Generation != 8 {
		t.Fatalf("first stats message = %#v, want generation 8 stats event", statsMsg)
	}
	cancelStats()
}
