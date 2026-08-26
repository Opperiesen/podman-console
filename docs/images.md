# Image management

Podman Console manages images on the currently active Podman connection. The feature stays
deliberately narrow: inspect local images, pull one registry reference, and remove one exact image
after confirmation.

## Browse and inspect

From the container inventory, press `i` to open the Images view. The inventory shows references,
short IDs, an available digest, local size, creation time, and whether the image is dangling or
read-only. Press `/` to filter locally by repository, tag, ID, or digest; filtering does not issue
another request to the Podman host.

Select an image with `↑`/`k` and `↓`/`j`, then press `Enter` for authoritative details. The details
view keeps the full image ID, all tags and digests, labels, creation metadata, and the number of
containers reported by the host.

## Pull

Press `P` from the Images view or image details, enter one non-empty registry reference, and press
`Enter`. Podman Console displays progress fragments in arrival order and keeps the reference and
active target visible. A successful pull refreshes the image inventory from Podman; the application
does not invent a local row from the pull response.

Press `Esc` during a pull to cancel its request context. Received progress remains visible, and the
inventory is refreshed so the final host state—not the client’s assumption—decides whether an image
exists. Registry authentication, certificates, signature policy, and credentials remain entirely
owned by the configured Podman service. Podman Console never asks for or stores a registry password.

## Safe removal

Press `D` on one selected image. The confirmation names the active host, the selected reference,
and the full image ID. `Esc`/`n` cancels without sending a mutation request; `Enter`/`y` sends one
exact image identity with force, all-images, ignore, and prune behavior disabled.

An image used by a container is not force-removed. Podman’s error is shown and the stable inventory
is preserved. A successful removal is followed by an authoritative refresh.

## Validation

The default tests use deterministic fakes and do not need Podman or a registry. To opt into the
live image workflow, provide a disposable reference that does not already exist on the target:

```sh
PODMAN_CONSOLE_URI='ssh://user@host/run/user/1000/podman/podman.sock' \
PODMAN_CONSOLE_IDENTITY='/path/to/identity' \
PODMAN_CONSOLE_TEST_IMAGE='quay.io/example/podman-console-test:latest' \
go test -tags=containers_image_openpgp,remote,integration -run TestLiveImageWorkflow -v ./tests/integration
```

The test refuses to remove a pre-existing matching image and removes the image it pulled before it
finishes.
