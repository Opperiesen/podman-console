package app

import (
	"errors"
	"testing"
	"time"

	"github.com/Opperiesen/podman-console/internal/domain"
	"github.com/Opperiesen/podman-console/internal/ui"
)

func TestStatsPreserveLastSampleWhenStreamFails(t *testing.T) {
	model, _ := testModel(t)
	model.screen = ui.ScreenStats
	model.streamGeneration = 3
	sample := domain.ContainerStats{ContainerID: "abcdef", CPUPercent: 12.5, MemoryUsageBytes: 1024, ObservedAt: time.Unix(100, 0)}
	updated, cmd := model.Update(statsStreamEvent{Generation: 3, Sample: &sample, Next: make(chan statsStreamEvent)})
	model = updated.(*Model)
	if cmd == nil || model.stats == nil || model.stats.CPUPercent != sample.CPUPercent {
		t.Fatalf("sample state = %#v command:%v", model.stats, cmd)
	}
	updated, cmd = model.Update(statsStreamEvent{Generation: 3, Done: true, Err: errors.New("stats unavailable")})
	model = updated.(*Model)
	if cmd != nil || !model.streamStopped || model.stats == nil || model.stats.MemoryUsageBytes != 1024 || model.err == nil {
		t.Fatalf("failed stats state = sample:%#v stopped:%v err:%v command:%v", model.stats, model.streamStopped, model.err, cmd)
	}
}
