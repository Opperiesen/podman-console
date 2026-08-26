# Image Client Contract

The image feature extends the existing `PodmanClient` abstraction. The UI and application model
must not import Podman binding packages.

## List

- Return image summaries for the active connection.
- Preserve host errors as typed adapter errors.
- Return an empty collection without treating it as a failure.

## Inspect

- Accept one image ID or unambiguous reference.
- Return authoritative details for that exact identity.
- Surface not-found, authorization, transport, and host errors without substitution.

## Pull

- Accept one validated non-empty registry image reference.
- Emit ordered progress events tied to the connection and reference.
- Respect context cancellation and close the stream on request completion.
- Never receive or persist a password from the application layer.
- Return reported image IDs only as operation feedback; the model still refreshes the inventory.

## Remove

- Accept exactly one image identity.
- Force, all, ignore, and prune options MUST remain disabled.
- Return deleted and untagged results plus per-image errors when supplied by the host.
- Respect context cancellation before the mutation is sent.

## Safety and stale responses

- Every asynchronous request captures the active connection identity.
- Every image mutation captures the full or unambiguous image identity shown in the confirmation.
- Late responses for another connection, selection, or operation are ignored and cannot alter the
  visible target or trigger a refresh for the wrong host.
