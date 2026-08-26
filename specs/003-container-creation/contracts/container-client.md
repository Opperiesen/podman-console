# Container Creation Client Contract

The feature extends the existing `PodmanClient` abstraction. The UI and application model must not
import Podman binding packages.

## Run one container

Add a transport-neutral operation equivalent to:

```go
RunContainer(ctx context.Context, request domain.ContainerCreateRequest) (domain.ContainerRunResult, error)
```

The adapter MUST:

- reject an empty image ID or invalid request before contacting Podman;
- build a `specgen.SpecGenerator` from the captured full image ID;
- set only the explicit name and optional argument list;
- create exactly one container with `containers.CreateWithSpec`;
- return the exact create ID and warnings;
- call `containers.Start` exactly once for that ID;
- return `Started: true` only after start succeeds;
- return the created ID with `Started: false` when start fails or is cancelled after creation;
- wrap errors with the relevant create/start action and typed category;
- never force, replace, remove, attach, pull, or retry implicitly.

## Payload boundary

The generated create payload must not contain operator credentials or unsupported configuration.
Interactive stdin and terminal attachment are disabled; the application does not receive a stream
from create/start.

## Cancellation

Cancellation before the create request means zero host mutations. Cancellation after the host has
returned a create ID may leave the container created but not started; that ID must be returned and
shown so the operator can make the next decision.

## Refresh

The model refreshes container and image inventories after both successful and partial outcomes. It
uses the host’s responses as the source of truth and never appends a guessed row.

## Stale responses

The caller captures the active connection identity, selected image ID, and operation generation.
The adapter need not know UI generations; the model ignores messages that no longer match them.
