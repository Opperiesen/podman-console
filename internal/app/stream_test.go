package app

import (
	"errors"
	"testing"

	"github.com/Opperiesen/podman-console/internal/domain"
	"github.com/Opperiesen/podman-console/internal/ui"
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
