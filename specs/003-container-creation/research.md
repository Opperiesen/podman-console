# Research: Run a Container from a Local Image

## Existing boundaries

The repository already separates domain values, the transport-neutral `PodmanClient`, the official
bindings adapter, asynchronous Bubble Tea commands, and UI rendering. The feature should add one
container-run request to those boundaries rather than letting the UI import Podman packages or
construct API payloads.

## Podman binding path

The pinned `go.podman.io/podman/v6@v6.1.0` module provides:

- `specgen.NewSpecGenerator(image, false)` to create a container specification from an image;
- `containers.CreateWithSpec(ctx, spec, nil)` returning a `ContainerCreateResponse` with the new ID
  and any warnings;
- `containers.Start(ctx, id, nil)` to start the created container.

The binding has no atomic create-and-start call in the existing API. The adapter therefore performs
the ordered two-step operation and returns the created ID when the second step fails. The model must
represent that partial outcome instead of pretending the sequence is transactional.

## Minimal payload

The adapter will set only the selected image ID, explicit name, and optional command. It will leave
environment, terminal, stdin, mounts, ports, networks, pods, privileges, restart policy, resource
limits, and replacement behavior unset or explicitly disabled. Starting through the binding without
an attach stream is the detached behavior required by this feature.

## Command semantics

The command field is intentionally an argument-only line. `strings.Fields`-style tokenization is
predictable and needs no new dependency, while shell operators and substitutions are rejected. The
image default command is used when the field is blank. This avoids silently turning a terminal form
into a remote shell.

## Safety and stale state

The form captures the image full ID, active profile identity, and image generation. Confirmation
checks that the same connection and image are still active. A late create/start result is accepted
only by the originating operation generation. A start failure after creation keeps the returned ID
visible, because deleting it automatically would be an unrequested second mutation.

## Testing strategy

- domain tests cover name and command validation;
- fake-client model tests cover no-request cancellation, exact identity, ordering, refresh, stale
  targets, and partial results;
- `httptest` adapter tests inspect the JSON create payload and assert the start request follows it;
- UI tests cover narrow layouts, read-only image identity, confirmation, validation, and feedback;
- opt-in integration creates one disposable container on the existing Rocky/Podman workflow and
  cleans it up by exact ID.

No new module dependency is required.
