package podman

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Opperiesen/podman-console/internal/domain"
	"github.com/opencontainers/go-digest"
	networktypes "go.podman.io/common/libnetwork/types"
	"go.podman.io/podman/v6/libpod/define"
	imageTypes "go.podman.io/podman/v6/pkg/domain/entities/types"
	"go.podman.io/podman/v6/pkg/inspect"
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
		if strings.HasSuffix(r.URL.Path, "/_ping") {
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

func TestConvertImagesPreservesReferencesDigestsAndInspectMetadata(t *testing.T) {
	created := time.Unix(1_700_000_000, 0)
	row := &imageTypes.ImageSummary{
		ID: "sha256:image", RepoTags: []string{"repo/app:latest", "repo/app:latest"},
		RepoDigests: []string{"repo/app@sha256:abc"}, Names: []string{"repo/app:latest", "repo/app:stable"},
		Created: created.Unix(), Size: 2048, Containers: 3, Dangling: true,
	}
	summary := convertImageSummary(row)
	if !reflect.DeepEqual(summary.References, []string{"repo/app:latest", "repo/app:stable"}) {
		t.Fatalf("references = %#v", summary.References)
	}
	if summary.Size != 2048 || summary.Containers != 3 || !summary.Dangling || !summary.CreatedAt.Equal(created) {
		t.Fatalf("summary = %#v", summary)
	}

	digestValue := digest.Digest("sha256:inspect")
	details := convertImageDetails(&inspect.ImageData{
		ID: "sha256:image", Digest: digestValue, RepoTags: []string{"repo/app:latest"},
		RepoDigests: []string{"repo/app@sha256:abc"}, Created: &created, Size: 4096,
		Parent: "sha256:parent", Architecture: "arm64", Os: "linux", Labels: map[string]string{"role": "web"},
	})
	if details.ID != "sha256:image" || details.ParentID != "sha256:parent" || details.Architecture != "arm64" || details.OS != "linux" || details.Labels["role"] != "web" {
		t.Fatalf("details = %#v", details)
	}
	if details.Digest != "sha256:inspect" || details.Size != 4096 {
		t.Fatalf("inspect identity = %#v", details.ImageSummary)
	}
}

func TestBindingsImageOperationsUseSafeOptionsAndOrderedPullEvents(t *testing.T) {
	var pullReference string
	var pullProgress []string
	var removeQuery map[string][]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/_ping") {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/images/json"):
			_ = json.NewEncoder(w).Encode([]map[string]any{{
				"Id": "sha256:image", "RepoTags": []string{"repo/app:latest"}, "RepoDigests": []string{"repo/app@sha256:abc"},
				"Created": int64(1_700_000_000), "Size": int64(2048), "Containers": 2,
			}})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/images/sha256:image/json"):
			_, _ = w.Write([]byte(`{"Id":"sha256:image","RepoTags":["repo/app:latest"],"RepoDigests":["repo/app@sha256:abc"],"Created":"2023-11-14T22:13:20Z","Size":2048,"Architecture":"arm64","Os":"linux","Labels":{"role":"web"}}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/images/pull"):
			pullReference = r.URL.Query().Get("reference")
			if r.URL.Query().Get("alltags") != "false" || r.URL.Query().Get("quiet") != "false" {
				t.Errorf("pull safety query = %v", r.URL.Query())
			}
			_, _ = w.Write([]byte(`{"stream":"layer-a\n"}
{"stream":"layer-b\n"}
{"images":["sha256:pulled"]}
`))
		case r.Method == http.MethodDelete && strings.HasSuffix(r.URL.Path, "/images/remove"):
			removeQuery = r.URL.Query()
			_, _ = w.Write([]byte(`{"Deleted":["sha256:image"],"Untagged":[],"Errors":[]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	profile := domain.ConnectionProfile{Name: "test", URI: "tcp://" + strings.TrimPrefix(server.URL, "http://")}
	client, err := (BindingsFactory{}).Connect(context.Background(), profile)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	images, err := client.ListImages(context.Background())
	if err != nil || len(images) != 1 || images[0].PrimaryReference() != "repo/app:latest" {
		t.Fatalf("ListImages() = %#v, %v", images, err)
	}
	details, err := client.InspectImage(context.Background(), "sha256:image")
	if err != nil || details.Architecture != "arm64" || details.Labels["role"] != "web" {
		t.Fatalf("InspectImage() = %#v, %v", details, err)
	}
	var events []domain.ImagePullEvent
	err = client.PullImage(context.Background(), "quay.io/example/app:latest", func(event domain.ImagePullEvent) {
		events = append(events, event)
		if event.Kind == domain.ImagePullProgress {
			pullProgress = append(pullProgress, event.Text)
		}
	})
	if err != nil || pullReference != "quay.io/example/app:latest" || strings.Join(pullProgress, "") != "layer-a\nlayer-b\n" {
		t.Fatalf("PullImage() = events:%#v err:%v reference:%q progress:%q", events, err, pullReference, pullProgress)
	}
	if len(events) != 3 || events[2].Kind != domain.ImagePullSuccess || len(events[2].ImageIDs) != 1 {
		t.Fatalf("pull events = %#v", events)
	}
	if err := client.RemoveImage(context.Background(), "sha256:image"); err != nil {
		t.Fatalf("RemoveImage() error = %v", err)
	}
	if removeQuery["images"][0] != "sha256:image" || removeQuery["all"][0] != "false" || removeQuery["force"][0] != "false" || removeQuery["ignore"][0] != "false" || removeQuery["lookupmanifest"][0] != "false" || removeQuery["noprune"][0] != "true" {
		t.Fatalf("remove safety query = %#v", removeQuery)
	}
}

func TestBindingsRunContainerBuildsMinimalPayloadAndStartsInOrder(t *testing.T) {
	var paths []string
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/_ping") {
			w.WriteHeader(http.StatusOK)
			return
		}
		paths = append(paths, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/containers/create"):
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode create payload: %v", err)
			}
			_, _ = w.Write([]byte(`{"Id":"created-id","Warnings":["using image default network"]}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/containers/created-id/start"):
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	profile := domain.ConnectionProfile{Name: "test", URI: "tcp://" + strings.TrimPrefix(server.URL, "http://")}
	client, err := (BindingsFactory{}).Connect(context.Background(), profile)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	result, err := client.RunContainer(context.Background(), domain.ContainerCreateRequest{
		ImageID: "sha256:image", ImageReference: "quay.io/example/app:latest", Name: "web", Command: []string{"sleep", "60"},
	})
	if err != nil {
		t.Fatalf("RunContainer() error = %v", err)
	}
	if result.ContainerID != "created-id" || !result.Started || len(result.Warnings) != 1 {
		t.Fatalf("RunContainer() result = %#v", result)
	}
	if !reflect.DeepEqual(paths, []string{"POST /v6.1.0/libpod/containers/create", "POST /v6.1.0/libpod/containers/created-id/start"}) {
		t.Fatalf("request order = %#v", paths)
	}
	if payload["image"] != "sha256:image" || payload["name"] != "web" {
		t.Fatalf("create identity payload = %#v", payload)
	}
	command, ok := payload["command"].([]any)
	if !ok || !reflect.DeepEqual(command, []any{"sleep", "60"}) {
		t.Fatalf("create command payload = %#v", payload["command"])
	}
	if payload["terminal"] != false || payload["stdin"] != false {
		t.Fatalf("interactive payload = %#v", payload)
	}
	for _, field := range []string{"env", "mounts", "portmappings", "networks", "privileged", "restart_policy", "pod"} {
		if _, exists := payload[field]; exists {
			t.Errorf("unsupported create field %q present in payload: %#v", field, payload[field])
		}
	}
}

func TestBindingsRunContainerReturnsPartialResultWhenStartFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/_ping") {
			w.WriteHeader(http.StatusOK)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/containers/create") {
			_, _ = w.Write([]byte(`{"Id":"created-id","Warnings":[]}`))
			return
		}
		if strings.HasSuffix(r.URL.Path, "/containers/created-id/start") {
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte("name is already in use"))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	profile := domain.ConnectionProfile{Name: "test", URI: "tcp://" + strings.TrimPrefix(server.URL, "http://")}
	client, err := (BindingsFactory{}).Connect(context.Background(), profile)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	result, err := client.RunContainer(context.Background(), domain.ContainerCreateRequest{ImageID: "sha256:image", Name: "web"})
	if result.ContainerID != "created-id" || result.Started || err == nil {
		t.Fatalf("partial RunContainer() = result:%#v err:%v", result, err)
	}
	var operation *domain.OperationError
	if !errors.As(err, &operation) || operation.Category != domain.ErrorPartial || operation.TargetID != "created-id" {
		t.Fatalf("partial error = %#v", err)
	}
}
