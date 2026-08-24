package podman

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Opperiesen/podman-console/internal/domain"
	networktypes "go.podman.io/common/libnetwork/types"
	"go.podman.io/podman/v6/libpod/define"
)

func TestBindingsFactoryConnectRejectsInvalidProfileWithoutHostAccess(t *testing.T) {
	factory := BindingsFactory{}
	_, err := factory.Connect(context.Background(), domain.ConnectionProfile{Name: "bad", URI: "http://example.test"})
	if err == nil {
		t.Fatal("Connect() returned nil for invalid profile")
	}
	opErr, ok := err.(*domain.OperationError)
	if !ok {
		t.Fatalf("Connect() error type = %T, want *domain.OperationError", err)
	}
	if opErr.Category != domain.ErrorInvalidConfig || opErr.Action != domain.ActionList {
		t.Fatalf("Connect() OperationError = %#v", opErr)
	}
}

func TestBindingsOperationContextHonorsCallerCancellation(t *testing.T) {
	base, baseCancel := context.WithCancel(context.Background())
	defer baseCancel()
	client := &bindingsClient{base: base}

	request, cancel := context.WithCancel(context.Background())
	opCtx, done := client.operationContext(request)
	defer done()
	cancel()

	select {
	case <-opCtx.Done():
	case <-time.After(time.Second):
		t.Fatal("operation context was not cancelled with the caller context")
	}
}

func TestConvertPorts(t *testing.T) {
	t.Parallel()

	input := []networktypes.PortMapping{{HostIP: "127.0.0.1", HostPort: 8080, ContainerPort: 80, Protocol: "tcp"}}
	want := []domain.PortMapping{{HostIP: "127.0.0.1", HostPort: 8080, ContainerPort: 80, Protocol: "tcp"}}
	if got := convertPorts(input); !reflect.DeepEqual(got, want) {
		t.Fatalf("convertPorts() = %#v, want %#v", got, want)
	}
	if got := convertPorts(nil); got == nil || len(got) != 0 {
		t.Fatalf("convertPorts(nil) = %#v, want non-nil empty slice", got)
	}
}

func TestConvertInspectPorts(t *testing.T) {
	t.Parallel()

	input := map[string][]define.InspectHostPort{
		"8080/tcp": {{HostIP: "127.0.0.1", HostPort: "18080"}, {HostIP: "::1", HostPort: "18081"}},
		"bad/udp":  {{HostIP: "0.0.0.0", HostPort: "not-a-port"}},
	}
	got := convertInspectPorts(input)
	if len(got) != 3 {
		t.Fatalf("convertInspectPorts() returned %d mappings, want 3: %#v", len(got), got)
	}

	want := map[domain.PortMapping]bool{
		{HostIP: "127.0.0.1", HostPort: 18080, ContainerPort: 8080, Protocol: "tcp"}: true,
		{HostIP: "::1", HostPort: 18081, ContainerPort: 8080, Protocol: "tcp"}:       true,
		{HostIP: "0.0.0.0", ContainerPort: 0, Protocol: "udp"}:                       true,
	}
	for _, mapping := range got {
		if !want[mapping] {
			t.Errorf("unexpected converted mapping %#v", mapping)
		}
		delete(want, mapping)
	}
	if len(want) != 0 {
		t.Errorf("missing converted mappings: %#v", want)
	}
}

func TestFirstNonEmpty(t *testing.T) {
	t.Parallel()

	if got := firstNonEmpty(" ", "", "image:latest"); got != "image:latest" {
		t.Fatalf("firstNonEmpty() = %q, want image:latest", got)
	}
	if got := firstNonEmpty(" ", "\t"); got != "" {
		t.Fatalf("firstNonEmpty(all empty) = %q, want empty", got)
	}
}

func TestBindingsLifecycleMapsSuccessAndStaleErrors(t *testing.T) {
	var paths []string
	status := http.StatusOK
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_ping" {
			w.WriteHeader(http.StatusOK)
			return
		}
		paths = append(paths, r.Method+" "+r.URL.Path)
		w.WriteHeader(status)
		if status == http.StatusNotFound {
			_, _ = w.Write([]byte("no such container"))
		} else if r.Method == http.MethodDelete {
			_, _ = w.Write([]byte("[]"))
		}
	}))
	defer server.Close()

	profile := domain.ConnectionProfile{Name: "test", URI: "tcp://" + strings.TrimPrefix(server.URL, "http://")}
	client, err := (BindingsFactory{}).Connect(context.Background(), profile)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	for _, test := range []struct {
		action domain.Action
		call   func(context.Context, string) error
		suffix string
	}{
		{action: domain.ActionStart, call: client.Start, suffix: "/start"},
		{action: domain.ActionStop, call: client.Stop, suffix: "/stop"},
		{action: domain.ActionRestart, call: client.Restart, suffix: "/restart"},
		{action: domain.ActionRemove, call: client.Remove, suffix: ""},
	} {
		status = http.StatusOK
		if err := test.call(context.Background(), "container-id"); err != nil {
			t.Errorf("%s success error = %v", test.action, err)
		}
		if test.suffix != "" && !strings.HasSuffix(paths[len(paths)-1], test.suffix) {
			t.Errorf("%s request = %q, want suffix %q", test.action, paths[len(paths)-1], test.suffix)
		}
	}

	status = http.StatusNotFound
	err = client.Stop(context.Background(), "missing")
	if err == nil {
		t.Fatal("Stop() returned nil for a missing container")
	}
	var operation *domain.OperationError
	if !errors.As(err, &operation) || operation.Category != domain.ErrorStaleTarget {
		t.Fatalf("Stop() error = %#v, want stale target operation error", err)
	}
}

func TestConsumeLogStreamReturnsWhenProducerCompletesWithoutClosingChannels(t *testing.T) {
	stdout := make(chan string)
	stderr := make(chan string)
	done := make(chan error, 1)
	go func() {
		stdout <- "first line"
		done <- nil
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var lines []domain.LogLine
	err := consumeLogStream(ctx, "container-id", stdout, stderr, done, func(line domain.LogLine) {
		lines = append(lines, line)
	})
	if err != nil {
		t.Fatalf("consume log stream: %v", err)
	}
	if len(lines) != 1 || lines[0].Text != "first line" || lines[0].Stream != "stdout" {
		t.Fatalf("unexpected log lines: %#v", lines)
	}
}
