# Podman Container Creation

## Local validation

From the repository root:

```sh
go test -tags=containers_image_openpgp,remote ./...
go test -race -tags=containers_image_openpgp,remote ./...
go vet -tags=containers_image_openpgp,remote ./...
go build -tags=containers_image_openpgp,remote ./cmd/podman-console
```

All commands must pass without a live Podman host. The default suite must remain deterministic.

## Live acceptance

Use a dedicated Podman host and a disposable image that is safe to run without credentials. Set:

```sh
PODMAN_CONSOLE_URI='ssh://user@example.test/run/user/1000/podman/podman.sock' \
PODMAN_CONSOLE_TEST_CREATE_IMAGE='quay.io/libpod/alpine:latest' \
PODMAN_CONSOLE_TEST_CREATE_NAME='podman-console-create-live-test' \
PODMAN_CONSOLE_TEST_CREATE_COMMAND='sleep 30' \
go test -tags=containers_image_openpgp,remote,integration \
  -run TestLiveContainerCreateWorkflow -v ./tests/integration
```

The opt-in test must:

1. refuse to use a pre-existing test container name;
2. ensure the disposable image is local without relying on Podman Console credentials;
3. create one container from the exact local image ID with a safe name and finite test command;
4. verify the container is present and running or has the host-reported expected state;
5. stop and remove only the exact created container ID;
6. verify the container is absent at the end and leave any pre-existing image untouched.

The test must skip when its environment is missing and must never target a production host.
