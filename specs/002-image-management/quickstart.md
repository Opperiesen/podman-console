# Quickstart: Podman Image Management

## Local validation

From the repository root:

```sh
go test -tags=containers_image_openpgp,remote ./...
go test -race -tags=containers_image_openpgp,remote ./...
go vet -tags=containers_image_openpgp,remote ./...
go build -tags=containers_image_openpgp,remote ./cmd/podman-console
```

All commands must pass without a live Podman host.

## Live acceptance

Use a disposable image reference on a dedicated host. The test must verify:

1. the Images view lists the fixture image and shows its repository/tag, ID, size, and state;
2. filtering and details do not mutate the host;
3. pulling a disposable tag shows ordered progress and refreshes the inventory;
4. cancelling a pull leaves the partial stream visible and does not claim success;
5. cancelling image removal sends no request;
6. confirming removal names the exact host and image and removes only that image;
7. the fixture image is absent at the end of the test.

The live test remains opt-in and must use the existing connection environment contract. It must not
use a user’s important image, credentials, registry policy, or production host.

## Validation record — 2026-08-26

- `go test -tags=containers_image_openpgp,remote ./...` passes;
- `go test -race -tags=containers_image_openpgp,remote ./...` passes;
- `go vet -tags=containers_image_openpgp,remote ./...` passes;
- Darwin/Linux/Windows amd64/arm64 cross-builds pass with the production build tags;
- the opt-in integration package compiles and skips cleanly when no live URI or disposable image
  reference is supplied;
- the opt-in image workflow passes against the dedicated Rocky Linux 9.8/arm64 VM with rootless
  Podman 5.8.2 using `quay.io/libpod/alpine:latest`; pull, authoritative list, inspect, removal,
  and post-test cleanup all pass;
- the existing live inventory and container workflow also pass against that VM, and the VM is
  returned to its stopped state with no test container or image remaining;
- fake-backed model, UI, and `httptest` adapter tests cover image listing, details, local filtering,
  ordered pull progress, cancellation, malformed/registry errors, stale targets, exact removal,
  and safe removal options;
- the dependency graph and `go.sum` are unchanged and `go mod verify` passes;
- version metadata was promoted to `0.2.0` on `main` and published as `v0.2.0` after these gates
  passed; the next feature branch owns the `0.3.0` promotion.
