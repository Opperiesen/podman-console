# Research: Podman Image Management

## Decision 1: Use the existing Podman image bindings

The pinned `go.podman.io/podman/v6` module already exposes `images.List`, `images.GetImage`,
`images.Pull`, and `images.Remove`. The feature should extend the existing `PodmanClient` port and
keep all binding calls inside `internal/podman/bindings.go`.

This preserves the MVP boundary: the application model consumes domain values and typed events,
not Podman response structs or transport details.

## Decision 2: Treat pull progress as a typed stream

The binding accepts a progress writer and decodes Podman image-pull reports containing stream text,
image IDs, progress, and errors. The adapter will write each report to an application-owned event
buffer and emit ordered domain messages. The UI will never parse terminal output or JSON.

The request context remains the cancellation boundary. On clean EOF, cancellation, malformed input,
or a transport error, the model closes the operation and preserves progress already received.

## Decision 3: Use host defaults for registry policy

The first image workflow accepts one image reference and delegates authentication, certificates,
TLS verification, and signature policy to the configured Podman service. Adding a password prompt,
auth-file editor, or policy editor would expand the security and cross-platform surface beyond the
next vertical slice.

## Decision 4: Remove only one exact image without force

Podman exposes force, all, ignore, and prune options. The feature will set only the single-image
identity and leave force, all, and prune disabled. An image in use is a useful actionable failure,
not a reason to silently escalate privileges or delete dependent containers.

## Decision 5: Refresh from the host after every mutation

As with the MVP container lifecycle, a successful pull or removal will trigger a new authoritative
image list. The UI must not infer tags, digests, sizes, or deletion results from a local optimistic
state update.
