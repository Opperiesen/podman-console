# Quickstart: Podman Console MVP

## Prerequisites

- Go 1.25.9 or newer
- A terminal with UTF-8 support
- For live validation only: a reachable Podman service and a connection URI

## Build and test without Podman

From the repository root:

```sh
go test ./...
go vet ./...
go build ./cmd/podman-console
```

The commands above must succeed without a live Podman host.

## Cross-platform build check

```sh
GOOS=darwin GOARCH=arm64 go build -o dist/podman-console-darwin-arm64 ./cmd/podman-console
GOOS=linux GOARCH=amd64 go build -o dist/podman-console-linux-amd64 ./cmd/podman-console
GOOS=windows GOARCH=amd64 go build -o dist/podman-console-windows-amd64.exe ./cmd/podman-console
```

The resulting artifacts are smoke-checked on their target platform before a release.

## Live connection setup

Create a profile from the application or provide a profile entry with a URI such as:

```text
unix:///run/user/1000/podman/podman.sock
ssh://user@example.test/run/user/1000/podman/podman.sock
```

The managed user must have a Podman service available. For a rootless Linux service, the
official Podman tutorial documents enabling `podman.socket` and configuring SSH access.

## Acceptance scenarios

1. Start with a fake or live host containing one running and one stopped container. Select the
   host and verify both rows show name, short ID, image, and state.
2. Open a container detail view. Verify identifiers, image, state, ports, mounts, and networks
   are visible, including explicit empty values.
3. Start the stopped container. Verify the operation result is shown and the inventory refreshes.
4. Request stop, restart, and remove on the running container. Verify each confirmation names
   the host, container name, and identifier; cancel at least one and verify no mutation request
   is sent.
5. Open logs and enable follow. Verify existing lines remain visible and new lines appear; end
   the stream and verify the UI says that live data stopped while preserving the lines.
6. Open metrics. Verify CPU/memory samples show an observation time and that a disconnected
   stream does not erase the last sample.
7. Run the complete default test suite on a machine without Podman installed.
