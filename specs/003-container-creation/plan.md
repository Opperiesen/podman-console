# Implementation Plan: Run a Container from a Local Image

**Branch**: `003-container-creation` | **Date**: 2026-08-26 | **Spec**: [spec.md](./spec.md)

## Summary

Extend Podman Console with a focused create-and-start workflow from one selected local image. The
workflow stays target-aware, detached, explicitly confirmed, and honest about the non-atomic
create/start boundary.

## Technical Context

- **Language/Version**: Go 1.26.7
- **UI**: Bubble Tea v2, Bubbles v2, Lip Gloss v2
- **Runtime**: Podman Go bindings v6.1.0 through the existing adapter
- **Build tags**: `containers_image_openpgp,remote`
- **Storage**: no new storage; the active Podman host remains authoritative
- **Testing**: Go tests, race detector, vet, `httptest`, deterministic fakes, opt-in integration
- **Platforms**: Darwin/Linux/Windows, amd64/arm64

## Design Constraints

- Keep Podman-specific request construction in `internal/podman`.
- Keep image/container identity and operation status in `internal/domain`.
- Keep form state, generation checks, asynchronous commands, and refresh orchestration in
  `internal/app`.
- Keep the UI dependent only on `ViewData` and domain values.
- Do not add a shell parser, registry client, database, daemon, or third-party dependency.
- Treat create and start as ordered but non-atomic; expose the created ID on start failure.

## Proposed Flow

```text
Images inventory
    │ select local image
    ▼
Create form (image ID read-only, name required, command optional)
    │ validate
    ▼
Exact confirmation (host + image ID + name + command/default)
    │ confirm
    ▼
Podman CreateWithSpec ──success──▶ Podman Start
    │                                  │
    │ create failure                    ├─ success → refresh containers + images
    │                                  └─ failure/cancel → partial feedback + refresh
    └─ no request on cancel/stale target
```

## Repository Changes

```text
internal/domain/container.go       request/result/status values
internal/domain/operation.go       create/run action and error categories
internal/podman/client.go          RunContainer port
internal/podman/bindings.go        specgen + CreateWithSpec + Start adapter
internal/podman/errors.go          name conflict, image missing, partial mapping
internal/app/messages.go           create/run messages and commands
internal/app/model.go              form, confirmation, generation, refresh state
internal/ui/keys.go                create binding on image screens
internal/ui/layout.go              create form and feedback rendering
internal/ui/container_components.go or new component file
tests/fixtures/fake_client.go      deterministic create/start behavior
tests/integration/container_create_test.go  opt-in live workflow
```

Tests stay beside the affected package where practical. The exact component filename may reuse an
existing UI component file if that keeps the small codebase coherent.

## Error and Refresh Rules

1. Validate locally before opening confirmation.
2. Re-check active connection and image generation before sending create.
3. On create failure, preserve the last stable inventories and show the typed error.
4. On create success, always retain the returned ID.
5. On start success, refresh containers and images before final success feedback.
6. On start failure/cancellation, refresh both inventories and show created-but-not-started with the
   exact ID; never auto-remove or retry.

## Validation Gates

- fake/model/UI tests pass without Podman;
- `httptest` proves image ID, name, command, disabled interactive behavior, and create/start order;
- race detector and vet pass;
- six cross-builds pass with production tags;
- opt-in live test passes on dedicated Rocky Linux and leaves no test container;
- no dependency or license change is introduced;
- release metadata moves to `0.3.0` only after all gates pass.
