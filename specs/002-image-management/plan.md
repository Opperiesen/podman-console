# Implementation Plan: Podman Image Management

**Branch**: `002-image-management` | **Date**: 2026-08-25 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/002-image-management/spec.md`

## Summary

Extend Podman Console with a focused image workflow for the active host: browse local images,
inspect one image, pull one registry reference with ordered progress, and remove one image behind
the existing exact-target confirmation contract. The feature must reuse the current connection,
transport, asynchronous message, refresh, error, and testing boundaries.

## Technical Context

**Language/Version**: Go 1.26.7 development toolchain; repository module remains compatible with
the Go version declared in `go.mod`.

**Existing Dependencies**:

- `go.podman.io/podman/v6/pkg/bindings/images` for list, inspect, pull, and remove operations;
- Bubble Tea v2, Bubbles v2, and Lip Gloss v2 for the TUI;
- Go standard library for contexts, validation, ordered stream buffering, and tests.

**Existing Integration Points**:

- `internal/podman/client.go` is the transport-neutral port to extend;
- `internal/podman/bindings.go` is the only production binding adapter;
- `internal/domain` owns values shown by the UI;
- `internal/app` owns asynchronous state, target invalidation, and authoritative refresh;
- `internal/ui` owns screen layout and key bindings;
- deterministic fake clients and `httptest` services remain the default test boundary.

**Binding Decisions**:

- Use Podman bindings `images.List`, `images.GetImage`, `images.Pull`, and `images.Remove`.
- Route pull output through an `io.Writer` owned by the adapter so the application receives typed
  progress events rather than parsing human output.
- Call removal with force disabled and pass one exact image identity at a time.
- Let Podman resolve registry credentials, auth files, TLS, and signature policy; do not add
  credential fields to the application configuration.

**Testing**: Go testing, race detector, vet, fake client behavior tests, binding contract tests,
UI rendering tests, and an opt-in live test against one disposable image reference.

**Target Platform**: macOS, Linux, and Windows terminals; local or remote Podman service.

**Constraints**: No new dependency, no daemon, no database, no password storage, no force or bulk
image operation, and no live service requirement for the default suite.

## Constitution Check

| Principle | Check | Result |
|---|---|---|
| Operator-First Experience | Images, details, progress, target visibility, and keyboard hints are explicit | PASS |
| Native Podman Integration | Official image bindings remain behind `PodmanClient` | PASS |
| Safe-by-Default Operations | Removal is exact-target, confirmed, non-force, and followed by refresh | PASS |
| Cross-Platform, Single-Binary Delivery | No new runtime or platform-specific dependency | PASS |
| Small, Tested Increments | Three independently testable stories with fake-backed acceptance paths | PASS |

No constitution violation or complexity exception is required.

## Project Structure

### Documentation

```text
specs/002-image-management/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── image-client.md
└── checklists/
    └── requirements.md
```

### Source Changes

```text
internal/domain/image.go
internal/podman/client.go
internal/podman/bindings.go
internal/podman/errors.go
internal/app/messages.go
internal/app/model.go
internal/ui/keys.go
internal/ui/layout.go
internal/ui/components.go
tests/fixtures/fake_client.go
tests/integration/image_test.go
docs/images.md
```

The exact file split may follow the existing package conventions during implementation, but image
binding calls must remain isolated from the UI.

## Delivery Phases

### Phase 1: Foundation

Define image domain values and extend the client port and fake with list, inspect, pull events, and
remove behavior. Add typed image errors and target-aware operation messages before changing the UI.

### Phase 2: User Story 1 — Browse images

Implement the Images view, local filtering, empty/loading/error states, image details, refresh, and
navigation without mutations. Stop at the story checkpoint and validate against fake and `httptest`
hosts.

### Phase 3: User Story 2 — Pull image

Add the image-reference input, validation, cancellable ordered progress, completion/error status,
and authoritative refresh. Validate partial output and malformed/disconnected streams.

### Phase 4: User Story 3 — Remove image

Add exact-target confirmation, non-force removal, cancellation invalidation, in-use feedback, and
authoritative refresh. Validate that cancellation sends no request and stale targets cannot mutate.

### Phase 5: Polish and live acceptance

Document image behavior, add opt-in disposable-image integration coverage, run formatting, tests,
race, vet, six-target builds, and the existing Rocky Linux live workflow. Record release blockers
before tagging the next version.

## Complexity Tracking

No complexity exception is planned. The feature deliberately excludes image build, push, search,
auth, prune, bulk operations, and multi-host aggregation so it can remain a small vertical slice.
