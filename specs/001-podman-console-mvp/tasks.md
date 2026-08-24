# Tasks: Podman Console MVP

**Input**: Design documents from `specs/001-podman-console-mvp/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`,
`quickstart.md`

**Tests**: Included because the specification requires a deterministic default test suite and
acceptance coverage for connection, rendering, safety confirmation, and stream failure behavior.

**Organization**: Tasks are grouped by user story so each slice can be implemented and tested
independently after the foundational layer.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Establish the Go module, executable entry point, project guidance, and CI skeleton.

- [ ] T001 Initialize the Go module and pinned direct dependencies in `go.mod` and `go.sum`
- [ ] T002 [P] Create the executable entry point and version metadata in `cmd/podman-console/main.go`
- [ ] T003 [P] Add repository ignores for Go artifacts and local configuration in `.gitignore`
- [ ] T004 [P] Document project commands and validation gates in `AGENTS.md`
- [ ] T005 [P] Add baseline cross-platform formatting, test, vet, and build workflow in `.github/workflows/ci.yml`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Build the domain, configuration, Podman adapter, and UI runtime boundaries required
by every user story.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [ ] T006 [P] Define connection, container, stream, and operation domain values in `internal/domain/connection.go`, `internal/domain/container.go`, and `internal/domain/operation.go`
- [ ] T007 [P] Define the `PodmanClient` connection, inventory, lifecycle, log, and stats port in `internal/podman/client.go`
- [ ] T008 [P] Implement JSON connection-profile validation and platform config-path resolution in `internal/config/model.go` and `internal/config/store.go`
- [ ] T009 Implement the Podman bindings adapter and typed error classification in `internal/podman/bindings.go` and `internal/podman/errors.go`
- [ ] T010 [P] Add deterministic fake-client fixtures for inventories, details, mutations, logs, metrics, and failures in `tests/fixtures/fake_client.go`
- [ ] T011 [P] Define application messages, screen state, selection state, and cancellation state in `internal/app/messages.go` and `internal/app/model.go`
- [ ] T012 [P] Define shared key bindings, styles, responsive layout helpers, and common components in `internal/ui/keys.go`, `internal/ui/styles.go`, `internal/ui/layout.go`, and `internal/ui/components.go`
- [ ] T013 Add constructor and process wiring for config store, Podman client factory, and Bubble Tea program in `internal/app/app.go` and `cmd/podman-console/main.go`
- [ ] T014 [P] Add unit tests for domain validation, profile persistence, error classification, and fake-client behavior in `internal/domain/*_test.go`, `internal/config/*_test.go`, and `internal/podman/*_test.go`

**Checkpoint**: The project builds, the app model can run without a live host, and all transport
calls are behind a deterministic contract.

---

## Phase 3: User Story 1 - Connect and See Containers (Priority: P1) 🎯 MVP

**Goal**: Select a local or remote profile and render a responsive, refreshable container
inventory with explicit empty and error states.

**Independent Test**: Start the app with the fake client, select a reachable profile, verify rows
for running and stopped containers, switch profiles, refresh, and exercise unreachable and empty
states without a live Podman host.

### Tests for User Story 1

- [ ] T015 [P] [US1] Add model tests for profile selection, loading, empty inventory, filtering, and refresh messages in `internal/app/model_test.go`
- [ ] T016 [P] [US1] Add rendering tests for target header, container rows, progress, empty state, and connection errors in `internal/ui/layout_test.go`

### Implementation for User Story 1

- [ ] T017 [US1] Implement profile loading, selection, and connection-switch commands in `internal/app/app.go` and `internal/app/model.go`
- [ ] T018 [US1] Implement the connection selector view with active-target indication and profile actions in `internal/ui/components.go` and `internal/app/model.go`
- [ ] T019 [US1] Implement asynchronous inventory loading, refresh, filtering, and stale-response protection in `internal/app/messages.go` and `internal/app/model.go`
- [ ] T020 [US1] Render the responsive inventory screen with target status, columns, key hints, loading state, empty state, and typed errors in `internal/ui/layout.go` and `internal/ui/styles.go`
- [ ] T021 [US1] Add the first-run profile configuration flow and save/remove profile behavior in `internal/config/store.go` and `internal/app/model.go`
- [ ] T022 [US1] Update the executable startup path and sample configuration documentation for the inventory workflow in `cmd/podman-console/main.go` and `docs/connections.md`

**Checkpoint**: User Story 1 is independently usable with a fake host and can be manually
validated against a reachable Podman service.

---

## Phase 4: User Story 2 - Inspect and Operate a Container (Priority: P2)

**Goal**: Open a selected container, inspect its metadata, and perform lifecycle actions only
after exact-target confirmation.

**Independent Test**: Select a fake container, open details, start/stop/restart/remove it, cancel
one destructive action, and verify success, refresh, stale-target, authorization, and failure
messages.

### Tests for User Story 2

- [ ] T023 [P] [US2] Add detail-view model tests for metadata, empty fields, stale selections, and target changes in `internal/app/detail_test.go`
- [ ] T024 [P] [US2] Add safety-dialog tests proving stop, restart, and remove require exact-target confirmation and cancellation sends no mutation in `internal/app/safety_test.go`
- [ ] T025 [P] [US2] Add adapter contract tests for start, stop, restart, and remove response/error mapping in `internal/podman/bindings_test.go`

### Implementation for User Story 2

- [ ] T026 [US2] Implement asynchronous container inspection and detail-domain translation in `internal/app/model.go` and `internal/podman/bindings.go`
- [ ] T027 [US2] Render container details, ports, mounts, networks, and explicit unavailable values in `internal/ui/layout.go` and `internal/ui/components.go`
- [ ] T028 [US2] Implement lifecycle commands and post-operation authoritative refresh in `internal/app/messages.go` and `internal/app/model.go`
- [ ] T029 [US2] Implement target-aware confirmation dialogs and cancellation invalidation in `internal/app/model.go`, `internal/ui/components.go`, and `internal/ui/keys.go`
- [ ] T030 [US2] Map stale-target, authorization, transport, and host errors to actionable operation feedback in `internal/podman/errors.go` and `internal/ui/layout.go`

**Checkpoint**: User Stories 1 and 2 both work independently; every mutation is visible,
confirmed when required, and followed by a host-authoritative refresh.

---

## Phase 5: User Story 3 - Read Logs and Resource Activity (Priority: P3)

**Goal**: Follow container logs and observe CPU/memory activity while preserving partial data and
clearly marking streams that are no longer current.

**Independent Test**: Open fake log and metrics streams, receive multiple updates, cancel or fail
the stream, and verify that existing data remains visible with a stopped/current indicator.

### Tests for User Story 3

- [ ] T031 [P] [US3] Add stream-model tests for log ordering, follow updates, cancellation, EOF, and partial-error preservation in `internal/app/stream_test.go`
- [ ] T032 [P] [US3] Add stats-model tests for CPU/memory samples, timestamps, polling cancellation, and unavailable values in `internal/app/stats_test.go`
- [ ] T033 [P] [US3] Add viewport rendering tests for log overflow, narrow terminals, and stopped streams in `internal/ui/stream_test.go`

### Implementation for User Story 3

- [ ] T034 [US3] Implement cancellable log iteration and typed stream messages in `internal/podman/bindings.go` and `internal/app/messages.go`
- [ ] T035 [US3] Implement log screen, follow toggle, viewport navigation, and partial-data status in `internal/app/model.go` and `internal/ui/layout.go`
- [ ] T036 [US3] Implement cancellable stats polling and last-sample preservation in `internal/podman/bindings.go` and `internal/app/model.go`
- [ ] T037 [US3] Render metrics cards, observation time, unavailable fields, and stopped-stream feedback in `internal/ui/layout.go` and `internal/ui/components.go`

**Checkpoint**: All three user stories are independently demonstrable with deterministic fake
streams and optional live Podman validation.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Make the project distributable, documented, and repeatably verifiable.

- [ ] T038 [P] Document connection setup, supported URI forms, credential boundaries, and troubleshooting in `docs/connections.md`
- [ ] T039 [P] Document global and container keyboard bindings, confirmation behavior, and terminal sizing in `docs/keybindings.md`
- [ ] T040 [P] Add user-facing project overview, screenshots or terminal recording guidance, build instructions, and scope boundaries in `README.md`
- [ ] T041 [P] Add opt-in live-host integration test harness and environment contract in `tests/integration/podman_test.go`
- [ ] T042 Run formatting, vet, unit tests, cross-platform builds, and quickstart acceptance checks; record any release blockers in `quickstart.md`
- [ ] T043 Review dependency licenses and binary behavior, then prepare the first versioned release metadata in `LICENSE`, `.github/workflows/ci.yml`, and `cmd/podman-console/main.go`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; can start immediately.
- **Foundational (Phase 2)**: Depends on Setup and blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational; defines the MVP inventory slice.
- **User Story 2 (Phase 4)**: Depends on Foundational and reuses the active selection from US1.
- **User Story 3 (Phase 5)**: Depends on Foundational and reuses the selected container from US1/US2.
- **Polish (Phase 6)**: Depends on the desired user stories being complete.

### User Story Dependencies

- **US1 (P1)**: No story dependency after Foundational.
- **US2 (P2)**: Uses US1 selection and refresh behavior, but its detail and mutation tests can run
  against a constructed model.
- **US3 (P3)**: Uses a selected container, but stream tests can run against a constructed model.

### Parallel Opportunities

- T002–T005 can run in parallel after module initialization.
- T006–T008, T010–T012, and T014 can run in parallel before T009/T013 integration.
- T015 and T016 can run in parallel; T017–T022 are sequential around the shared app model.
- T023–T025 can run in parallel before T026–T030.
- T031–T033 can run in parallel before T034–T037.
- T038–T041 can run in parallel after all stories; T042 and T043 remain final validation tasks.

## Parallel Example: User Story 1

```text
Task T015: model tests for profile selection and inventory states
Task T016: rendering tests for inventory states
```

After the tests exist, T017 and T018 can be developed in sequence, followed by T019–T022 because
they share the application model and persistence flow.

## Implementation Strategy

### MVP First

1. Complete Setup and Foundational phases.
2. Complete US1 and stop at its checkpoint.
3. Run fake-backed tests and a manual live-host inventory check.
4. Tag or demo the inventory-only MVP before adding mutations.

### Incremental Delivery

1. Add US2 and validate the safety contract independently.
2. Add US3 and validate partial stream behavior independently.
3. Complete documentation, live opt-in tests, and cross-platform artifacts.

## Notes

- Every task includes a concrete path and follows the required checkbox/ID/story-label format.
- Tests are written before the corresponding story implementation and use the fake client by
  default.
- A task that needs a live Podman service must remain opt-in and must not block `go test ./...`.
