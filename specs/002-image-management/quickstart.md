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
