# Tasks: Podman Image Management

**Input**: Design documents from `specs/002-image-management/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/image-client.md`

**Tests**: Required for this feature. Write the fake-backed test before each corresponding
implementation slice and keep live validation opt-in.

## Phase 1: Foundation

**Purpose**: Extend the domain and transport boundaries without changing the existing container
workflow.

- [ ] T001 [P] Add image domain values, operation status, and display helpers in `internal/domain/image.go`.
- [ ] T002 Extend the `PodmanClient` port with image list, inspect, pull-event, and single-remove methods in `internal/podman/client.go`.
- [ ] T003 [P] Extend typed Podman error classification for registry, image-not-found, image-in-use, and pull-stream failures in `internal/podman/errors.go`.
- [ ] T004 Implement image binding calls and ordered pull progress translation in `internal/podman/bindings.go`.
- [ ] T005 [P] Extend the deterministic fake client with image fixtures, pull events, cancellation, removal, and stale-target responses in `tests/fixtures/fake_client.go`.
- [ ] T006 Add image messages and target-aware operation state in `internal/app/messages.go` and `internal/app/model.go`.

**Checkpoint**: The client contract, fake behavior, and message vocabulary can represent all three
stories without importing Podman bindings into the UI.

## Phase 2: User Story 1 — Browse local images (P1)

**Goal**: Read-only inventory, filtering, details, refresh, and explicit empty/error states.

**Independent Test**: A fake host with tagged, dangling, and empty inventories passes all view and
model tests without a live service.

### Tests

- [ ] T007 [P] [US1] Add image list model tests for loading, empty state, refresh, filtering, and stale responses in `internal/app/image_test.go`.
- [ ] T008 [P] [US1] Add image detail model tests for tags, digests, labels, missing values, and stale selection in `internal/app/image_detail_test.go`.
- [ ] T009 [P] [US1] Add inventory and detail rendering tests for narrow terminals, long references, and selected rows in `internal/ui/image_test.go`.

### Implementation

- [ ] T010 [US1] Add Images navigation and key binding without conflicting with existing container, logs, metrics, or profile keys in `internal/ui/keys.go` and `internal/app/model.go`.
- [ ] T011 [US1] Implement image inventory loading, local filtering, refresh, and empty/error state transitions in `internal/app/model.go`.
- [ ] T012 [US1] Implement image detail loading and target invalidation in `internal/app/model.go`.
- [ ] T013 [US1] Render the responsive image inventory, details, target status, and key hints in `internal/ui/layout.go` and `internal/ui/components.go`.

**Checkpoint**: An operator can browse and inspect images without any host mutation.

## Phase 3: User Story 2 — Pull an image with visible progress (P2)

**Goal**: Validate one reference, show ordered progress, support cancellation, and refresh after
success.

**Independent Test**: A fake pull stream covering progress, completion, partial error, malformed
event, cancellation, and EOF produces the expected model state.

### Tests

- [ ] T014 [P] [US2] Add pull model tests for validation, ordered progress, success, partial error, cancellation, EOF, and stale target in `internal/app/image_pull_test.go`.
- [ ] T015 [P] [US2] Add pull rendering tests for input, progress, stopped stream, and actionable error feedback in `internal/ui/image_pull_test.go`.
- [ ] T016 [P] [US2] Add binding contract tests for image pull request options and progress translation in `internal/podman/bindings_test.go`.

### Implementation

- [ ] T017 [US2] Add image-reference input, validation, and pull commands in `internal/app/model.go` and `internal/ui/components.go`.
- [ ] T018 [US2] Implement cancellable pull event delivery and ordered progress state in `internal/app/messages.go` and `internal/app/model.go`.
- [ ] T019 [US2] Render pull progress, completion, partial output, cancellation, and errors without blocking the event loop in `internal/ui/layout.go`.
- [ ] T020 [US2] Trigger authoritative image refresh after a successful pull and preserve the last stable inventory on failure in `internal/app/model.go`.

**Checkpoint**: An operator can pull one image and understand its result without leaving the TUI.

## Phase 4: User Story 3 — Remove one image safely (P3)

**Goal**: Exact-target, non-force removal with confirmation, cancellation safety, error feedback,
and refresh.

**Independent Test**: The model proves cancellation sends no request and confirmation sends exactly
one non-force request for the captured image identity.

### Tests

- [ ] T021 [P] [US3] Add removal safety tests for exact target, cancel, confirm, stale selection, in-use, not-found, and authorization errors in `internal/app/image_remove_test.go`.
- [ ] T022 [P] [US3] Add binding contract tests proving force, all, ignore, and prune remain disabled in `internal/podman/bindings_test.go`.
- [ ] T023 [P] [US3] Add removal dialog and feedback rendering tests in `internal/ui/image_remove_test.go`.

### Implementation

- [ ] T024 [US3] Add image-specific target-aware confirmation content and key handling in `internal/app/model.go` and `internal/ui/components.go`.
- [ ] T025 [US3] Implement one-image non-force removal and typed outcome mapping in `internal/podman/bindings.go` and `internal/app/messages.go`.
- [ ] T026 [US3] Invalidate stale confirmations, prevent duplicate submits, and refresh the authoritative inventory after success in `internal/app/model.go`.
- [ ] T027 [US3] Render in-use, not-found, authorization, and successful removal feedback in `internal/ui/layout.go`.

**Checkpoint**: An operator can remove one image safely, and cancellation or a stale target cannot
mutate the host.

## Phase 5: Polish and live validation

- [ ] T028 [P] Document image navigation, pull credential boundaries, and safe removal behavior in `docs/images.md`.
- [ ] T029 [P] Add opt-in live image integration coverage using one disposable image reference in `tests/integration/image_test.go`.
- [ ] T030 [P] Update the README feature overview and quickstart links for image management.
- [ ] T031 Run formatting, default tests, race tests, vet, six-target builds, and the live acceptance workflow; record results in `specs/002-image-management/quickstart.md`.
- [ ] T032 Review dependency/license impact and prepare version metadata for the next release only after all stories and validation gates pass.

## Dependencies & Execution Order

- Phase 1 blocks all user stories.
- User Story 1 is independent and should land first because it provides the image navigation and
  refresh surface used by the later stories.
- User Story 2 depends on the image inventory and stream vocabulary from Phase 1, but not on image
  removal.
- User Story 3 depends on image selection and the existing safety contract, but can be tested
  independently from pull.
- Phase 5 depends on the desired stories being complete.
- Tasks marked `[P]` touch separate tests or documentation and can run in parallel when their
  prerequisites are complete.

## Incremental Delivery

1. Complete Phase 1 and validate the foundation.
2. Deliver US1 as a read-only image inventory checkpoint.
3. Add US2 and validate pull progress and cancellation.
4. Add US3 and validate exact-target removal.
5. Run live acceptance, document the result, and only then propose the next versioned release.
