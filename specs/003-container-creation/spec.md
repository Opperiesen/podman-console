# Feature Specification: Run a Container from a Local Image

**Feature Branch**: `003-container-creation`
**Created**: 2026-08-26
**Target Release**: `v0.3.0`
**Input**: Post-v0.2.0 follow-up for Podman Console.

## Overview

Podman Console can now inspect and pull local images, but it cannot turn one of those images into
a workload. This feature adds one deliberately narrow path: select an image already present on the
active Podman host, enter an explicit container name and an optional argument-only command, confirm
the exact target, then create and start one detached container.

The feature uses the existing active connection, client port, asynchronous model, authoritative
refreshes, and safety conventions. It does not become a general `podman run` front end.

### User Story 1 - Configure one container from a local image (Priority: P1)

As an operator, I want to select a local image and provide a name and optional command so that I can
prepare a repeatable container without leaving the terminal UI.

**Why this priority**: Image management is only useful for workloads when an operator can safely
turn one available image into a container.

**Independent Test**: Give a fake host one local image, open the create form from the Images view,
enter a valid name and command, and verify that the form shows the exact image and accepts the
configuration without contacting Podman.

**Acceptance Scenarios**:

1. **Given** a selected local image, **when** the operator opens the create form, **then** the
   active host, image reference, and full image ID are visible and the image identity is not
   editable.
2. **Given** an empty, whitespace-containing, or invalid name, **when** the operator submits the
   form, **then** the UI explains the validation failure and sends no create request.
3. **Given** a valid name and an optional command, **when** the operator edits the fields, **then**
   the UI shows the exact values that will be used and treats the command as argument tokens rather
   than evaluating it through a shell.

### User Story 2 - Create and start the confirmed container (Priority: P1)

As an operator, I want to confirm and start the configured container so that the new workload is
visible in the normal container workflow immediately.

**Why this priority**: The feature must complete the image-to-workload path rather than leave a
second manual command for the operator.

**Independent Test**: Confirm a valid fake request and verify one ordered create/start operation,
an authoritative container refresh, and an image usage refresh.

**Acceptance Scenarios**:

1. **Given** a valid configuration, **when** the operator confirms it, **then** the confirmation
   names the active host, full image ID, image reference, container name, and command/default-command
   behavior before any mutation is sent.
2. **Given** a confirmed configuration, **when** Podman accepts creation and start, **then** the UI
   reports success, refreshes containers and images authoritatively, and makes the new container
   available to inspect and operate.
3. **Given** an image whose default command exits quickly, **when** Podman accepts the start,
   **then** the operation still reports that start was accepted and the refreshed container state
   reflects the host-reported exit state.

### User Story 3 - Preserve safety and explain partial outcomes (Priority: P2)

As an operator, I want cancellation, stale selections, and failures to be explicit so that a
container is never created for an unintended image or host and a partial result is not hidden.

**Independent Test**: Cancel before confirmation, invalidate the image or connection before
confirmation, fail creation, and fail start after creation; verify the corresponding request count,
feedback, and refresh behavior.

**Acceptance Scenarios**:

1. **Given** the create form or confirmation, **when** the operator cancels, **then** no mutation
   request is sent and the prior inventory remains usable.
2. **Given** that the selected image or active connection changed after the form opened, **when** the
   operator confirms, **then** the operation is rejected as stale and no create request is sent.
3. **Given** that creation succeeds but start fails, **when** the operation ends, **then** the UI
   reports the exact created container ID as created-but-not-started, refreshes authoritative state,
   and does not silently remove or retry the container.
4. **Given** authorization, transport, image-not-found, or name-conflict errors, **when** the host
   rejects the operation, **then** the UI distinguishes the actionable category and preserves the
   last stable inventory.

## Edge Cases

- The active host has no images or the selected image disappears before confirmation; the create
  path is unavailable or becomes stale without sending a request.
- The selected image has no repository tag; the confirmation uses an explicit placeholder and the
  full image ID remains authoritative.
- The requested name conflicts with an existing container; Podman’s error is shown and no automatic
  rename is attempted.
- The optional command is blank; Podman uses the image’s configured entrypoint and command.
- The command contains shell operators, quotes, environment substitutions, or a NUL byte; the UI
  does not execute a shell and rejects unsupported input instead of guessing.
- The connection changes, disconnects, or is refreshed while create/start is in flight; late results
  cannot alter another target’s state.
- Creation succeeds and the client is cancelled before start; the created ID remains visible in
  feedback and the user can inspect or remove it through the existing workflow.
- The terminal is narrow or the command/reference is long; the form wraps or truncates for display
  without changing the submitted values.

## Functional Requirements

- **FR-001**: The system MUST provide a container-create view reachable from a selected local image
  on the active Podman connection.
- **FR-002**: The create request MUST capture the full image ID at form opening and MUST NOT accept
  an arbitrary remote reference or perform an implicit pull.
- **FR-003**: The UI MUST require a non-empty container name matching the supported safe-name grammar:
  1–63 characters, starting with an alphanumeric character, followed only by alphanumerics, `.`,
  `_`, or `-`.
- **FR-004**: The optional command MUST be represented as an ordered argument list. The UI MUST NOT
  invoke a shell, expand variables, interpret redirects, or persist credentials.
- **FR-005**: Before mutation, the confirmation MUST identify the active host, image reference,
  full image ID, container name, and command or explicit image-default behavior.
- **FR-006**: Cancelling the form or confirmation MUST send no create or start request.
- **FR-007**: After confirmation, the system MUST create exactly one detached container from the
  captured image ID and then issue exactly one start request for the returned container ID.
- **FR-008**: The implementation MUST use the official Podman Go bindings behind `PodmanClient` and
  MUST leave privileged mode, host environment, ports, mounts, networks, pods, restart policy,
  resource limits, replacement, and force behavior unset.
- **FR-009**: After a successful create/start sequence, the model MUST refresh both container and
  image inventories from the host and MUST NOT synthesize a guessed container or usage count.
- **FR-010**: If creation succeeds but start fails or is cancelled, the result MUST retain and show
  the exact created container ID, classify the operation as partial, refresh authoritative state,
  and avoid automatic deletion or retry.
- **FR-011**: Every asynchronous operation MUST capture the active connection identity and image
  generation; stale responses MUST be ignored and MUST NOT trigger a refresh for another target.
- **FR-012**: The system MUST classify validation, image-not-found, name-conflict, authorization,
  transport, and partial create/start errors into actionable feedback.
- **FR-013**: Registry authentication, certificates, and signature policy MUST remain owned by the
  configured Podman service; Podman Console MUST NOT collect or store passwords or private keys.
- **FR-014**: The default test suite MUST cover validation, confirmation cancellation, exact payload,
  create/start ordering, success refresh, stale targets, cancellation, and partial outcomes without
  requiring a live Podman host.
- **FR-015**: Live validation MUST be opt-in, use one disposable image and container name, and leave
  no container created solely by the test.

## Domain Vocabulary

- **Local image**: An image returned by the active host’s authoritative image inventory and captured
  by its full ID.
- **Create configuration**: The captured image ID, display reference, explicit container name, and
  optional ordered command arguments.
- **Detached start**: A start request that does not attach the TUI to the container’s stdin, stdout,
  or terminal; logs remain available through the existing logs view.
- **Partial outcome**: A create request returned a container ID, but the subsequent start request did
  not complete successfully.

## Success Criteria

- **SC-001**: On a fake host, 100% of valid confirmations send one create followed by one start for
  the captured image ID and never use a guessed image reference.
- **SC-002**: 100% of cancellation and stale-target tests send zero create requests.
- **SC-003**: Every acceptance confirmation names the exact active host, image ID, and container
  name before mutation.
- **SC-004**: A successful run refreshes the authoritative container and image inventories before
  reporting the workflow complete.
- **SC-005**: A create-success/start-failure test exposes the created ID and does not silently delete
  or retry it.
- **SC-006**: The opt-in Rocky Linux workflow creates, starts, observes, stops, and removes one
  disposable container and leaves the host clean.
- **SC-007**: The existing six-target build matrix and all v0.2.0 container/image workflows remain
  green without a new runtime dependency.

## Assumptions and Scope Boundaries

- The operator already has a configured local or remote Podman service and at most one active host.
- The selected image is already local; pulling remains the responsibility of the v0.2.0 image
  workflow.
- One container is created at a time and it starts detached.
- Image build, push, search, save/load, signing, registry administration, environment variables,
  ports, mounts, volumes, networks, pods, privileged/security options, resource limits, restart
  policies, attach/exec, bulk operations, and multi-host orchestration are out of scope.
- The feature extends the current domain/client/model/UI boundaries without adding a daemon,
  database, shell parser, or runtime dependency.
