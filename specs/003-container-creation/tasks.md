# Tasks: Run a Container from a Local Image

**Input**: Design documents from `/specs/003-container-creation/`
**Prerequisites**: `001-podman-console-mvp` and `002-image-management` are released on `main`.

## Phase 1: Domain and transport foundation

**Purpose**: Add the request/result vocabulary without leaking Podman bindings into the UI.

- [x] T001 [P] Add `ContainerCreateRequest`, `ContainerRunResult`, validation helpers, and create
  operation statuses in `internal/domain/container.go`.
- [x] T002 Extend `PodmanClient` with the one-container run operation in `internal/podman/client.go`.
- [x] T003 [P] Add create, name-conflict, image-not-found, and partial-start error classification in
  `internal/domain/operation.go` and `internal/podman/errors.go`.
- [x] T004 Implement `specgen.NewSpecGenerator`, `containers.CreateWithSpec`, and ordered
  `containers.Start` translation in `internal/podman/bindings.go`.
- [x] T005 [P] Extend the deterministic fake client with exact create/start calls, results, errors,
  and cancellation behavior in `tests/fixtures/fake_client.go`.
- [x] T006 Add create/run messages and target-aware operation state in `internal/app/messages.go`
  and `internal/app/model.go`.

**Checkpoint**: The client contract and fake can represent creation, start, cancellation, and
partial results without changing existing image or container workflows.

## Phase 2: User Story 1 — configure one container

**Goal**: Open a form from a selected image and validate a safe, explicit request without mutation.

### Tests

- [x] T007 [P] [US1] Add domain tests for name grammar, blank command, argument tokenization, and
  shell-control rejection in `internal/domain/container_test.go`.
- [x] T008 [P] [US1] Add model tests for image identity capture, form editing, validation feedback,
  and no-request cancellation in `internal/app/container_create_test.go`.
- [x] T009 [P] [US1] Add form rendering tests for read-only image identity, long values, narrow
  terminals, and validation errors in `internal/ui/container_create_test.go`.

### Implementation

- [x] T010 [US1] Add image-to-create navigation and non-conflicting key bindings in
  `internal/ui/keys.go` and `internal/app/model.go`.
- [x] T011 [US1] Implement image generation capture, name/command form state, local validation, and
  stale selection invalidation in `internal/app/model.go`.
- [x] T012 [US1] Render the create form, field hints, and disabled/empty-image states in
  `internal/ui/layout.go` and the appropriate UI component file.

**Checkpoint**: An operator can prepare a valid request and cancel it without contacting Podman.

## Phase 3: User Story 2 — create and start

**Goal**: Confirm one exact request, execute create then start, and refresh authoritative state.

### Tests

- [x] T013 [P] [US2] Add model tests for exact confirmation content, ordered create/start, success
  feedback, and container/image refresh in `internal/app/container_create_test.go`.
- [x] T014 [P] [US2] Add `httptest` binding coverage for the create JSON payload, disabled
  interactive settings, returned ID, and start request order in `internal/podman/bindings_test.go`.
- [x] T015 [P] [US2] Add UI tests for confirmation, progress/status transitions, and successful
  result rendering in `internal/ui/container_create_test.go`.

### Implementation

- [x] T016 [US2] Implement confirmation capture and asynchronous create/start command delivery in
  `internal/app/messages.go` and `internal/app/model.go`.
- [x] T017 [US2] Refresh container and image inventories from the host after a successful run and
  select the returned container without synthesizing details in `internal/app/model.go`.
- [x] T018 [US2] Render exact target confirmation, creating/starting status, and success feedback in
  `internal/ui/layout.go` and the appropriate UI component file.

**Checkpoint**: A confirmed local image can become a visible detached container through one
non-blocking workflow.

## Phase 4: User Story 3 — safety and partial outcomes

**Goal**: Make stale, cancelled, unauthorized, and partially completed operations explicit.

### Tests

- [x] T019 [P] [US3] Add tests for stale connection/image generation, duplicate submit prevention,
  and cancellation before/after create in `internal/app/container_create_test.go`.
- [x] T020 [P] [US3] Add tests for image-not-found, name-conflict, authorization, transport, and
  start-after-create failures in `internal/podman/errors_test.go` and app tests.
- [x] T021 [P] [US3] Add partial-result and exact-ID feedback rendering tests in
  `internal/ui/container_create_test.go`.

### Implementation

- [x] T022 [US3] Preserve the created ID on start failure/cancellation and prevent automatic remove
  or retry in `internal/podman/bindings.go` and `internal/app/model.go`.
- [x] T023 [US3] Add target-aware confirmation invalidation, duplicate-submit guards, and typed
  feedback mappings in `internal/app/model.go`.
- [x] T024 [US3] Document the local-image requirement, argument-only command semantics, detached
  behavior, and partial outcomes in `docs/containers.md` and `docs/keybindings.md`.

**Checkpoint**: Cancellation or stale state cannot create an unintended container, and a partial
operation never claims success or silently cleans up the target.

## Phase 5: Documentation and live validation

- [ ] T025 [P] Add opt-in Rocky Linux container-creation coverage with exact cleanup in
  `tests/integration/container_create_test.go`.
- [x] T026 [P] Update the README feature overview, quickstart links, and keyboard workflow for
  creating containers from local images.
- [ ] T027 Run formatting, default tests, race tests, vet, six-target builds, and the opt-in live
  acceptance workflow; record results in `specs/003-container-creation/quickstart.md`.
- [ ] T028 Review dependency/license impact and prepare version metadata for `0.3.0` only after all
  stories and validation gates pass.

## Dependencies and Execution Order

- Phase 1 blocks all user stories.
- User Story 1 blocks the confirmation and mutation flow because it owns the request shape and
  image generation capture.
- User Story 2 depends on the form and client operation but can be validated before partial-error
  polish.
- User Story 3 hardens the same operation and must pass before live validation.
- T007–T009, T013–T015, T019–T021, and T025–T026 are independent within their phases when their
  implementation contracts are stable.
