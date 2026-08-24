# Feature Specification: Podman Console MVP

**Feature Branch**: `001-podman-console-mvp`

**Created**: 2026-08-24

**Status**: Draft

**Input**: User description: "Create an independent cross-platform terminal UI for administering Podman containers on local and remote hosts, with connection selection, container listing, detail inspection, lifecycle actions, live logs, basic resource metrics, and explicit safety confirmations."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Connect and See Containers (Priority: P1)

As an operator, I want to select a local or remote Podman host and immediately see its
containers, so that I can understand the current state of a machine without memorizing CLI
commands.

**Why this priority**: A trustworthy inventory is the foundation for every other operation and
already provides value even when no mutation is needed.

**Independent Test**: Start the application with a reachable Podman host containing running and
stopped containers, select that host, and verify that the inventory identifies each container,
its state, and its image.

**Acceptance Scenarios**:

1. **Given** a saved connection to a reachable host, **When** the operator starts the application,
   **Then** the active host and its container inventory are visible without entering a shell
   command.
2. **Given** several containers with different states, **When** the operator refreshes the
   inventory, **Then** each container shows its name, short identifier, image, and current state.
3. **Given** multiple saved connections, **When** the operator opens the connection selector,
   **Then** the selector identifies each target and clearly shows which one is active.
4. **Given** a host that cannot be reached, **When** the operator selects it, **Then** the
   application explains the failure and keeps the rest of the interface usable.

### User Story 2 - Inspect and Operate a Container (Priority: P2)

As an operator, I want to inspect a container and perform its normal lifecycle actions from the
same interface, so that routine administration is fast and the target of every action is clear.

**Why this priority**: Inventory without an actionable detail view does not replace the daily
operator workflow. This story turns the inventory into a useful administration tool.

**Independent Test**: Select one container on a reachable host, open its details, start or stop
it, and verify that the displayed state and operation result match the host.

**Acceptance Scenarios**:

1. **Given** a selected container, **When** the operator opens its details, **Then** the
   application shows its identifiers, image, state, ports, mounts, and network information.
2. **Given** a stopped container, **When** the operator starts it, **Then** the application
   reports the operation result and refreshes the displayed state.
3. **Given** a running container, **When** the operator requests a stop or restart, **Then** the
   application shows the exact target and requires explicit confirmation before acting.
4. **Given** an operation fails, **When** the host returns an error, **Then** the application
   displays an actionable error and does not present the operation as successful.
5. **Given** a container disappears or changes state outside the application, **When** the
   operator acts on the stale selection, **Then** the application refreshes the target and
   explains why the requested action could not be completed.

### User Story 3 - Read Logs and Resource Activity (Priority: P3)

As an operator, I want to follow a container's logs and observe its basic resource activity, so
that I can diagnose a problem without switching tools.

**Why this priority**: Logs and resource signals make the application useful during incidents,
but they depend on a reliable connection and container detail workflow.

**Independent Test**: Select a running container that emits output, open its logs, follow new
lines, and view resource activity while the container runs.

**Acceptance Scenarios**:

1. **Given** a container with existing output, **When** the operator opens its logs, **Then** the
   application shows recent lines in chronological order and indicates when no output is
   available.
2. **Given** log following is enabled, **When** the container emits new output, **Then** the
   new lines appear without restarting the view.
3. **Given** a container exposes resource activity, **When** the operator opens its metrics,
   **Then** the application displays current CPU and memory usage with a clear refresh time.
4. **Given** the log or metrics stream ends unexpectedly, **When** the host reports the failure,
   **Then** the application preserves already received data and explains that live updates
   stopped.

### Edge Cases

- A host may have no containers; the inventory must show an explicit empty state rather than a
  blank screen.
- A container may have a generated name, no published ports, no mounts, or no recent logs; the
  detail view must represent those values honestly.
- A remote connection may be valid but slow; the interface must show that a request is in
  progress and must not interpret a delay as a successful empty response.
- A container may be removed while its logs or metrics are open; the view must stop polling and
  explain that the target no longer exists.
- The terminal may be too small for every field; the application must preserve the target,
  state, and action feedback before optional fields.
- A destructive action may be cancelled; cancellation must leave the host unchanged.
- The host may return an authorization error; the application must distinguish it from a network
  failure and avoid retrying a mutation automatically.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The application MUST run as a terminal UI on macOS, Linux, and Windows.
- **FR-002**: The application MUST allow the operator to create, select, rename, and remove
  connection profiles without storing credentials in the project configuration by default.
- **FR-003**: The application MUST identify the active host in every view that can perform an
  operation.
- **FR-004**: The application MUST list containers for the active host with name, short
  identifier, image, and current state.
- **FR-005**: The application MUST allow the operator to refresh and filter the container
  inventory without losing the active host selection.
- **FR-006**: The application MUST display container identifiers, image, state, ports, mounts,
  and network information in a detail view.
- **FR-007**: The application MUST support starting, stopping, restarting, and removing a
  selected container.
- **FR-008**: The application MUST require explicit confirmation for stopping, restarting, and
  removing a container, and the confirmation MUST include the target name and identifier.
- **FR-009**: The application MUST display the result of every requested operation, including
  failures, without hiding host-provided error details.
- **FR-010**: The application MUST display recent container logs and support following new log
  output while the view is open.
- **FR-011**: The application MUST display current CPU and memory activity for a selected
  container when the host provides those measurements.
- **FR-012**: The application MUST preserve already received logs and metrics when a live update
  stream fails, and MUST identify that the data is no longer current.
- **FR-013**: The application MUST provide keyboard navigation, visible key hints, and a way to
  cancel an in-progress request or confirmation.
- **FR-014**: The application MUST keep read-only inspection available when an operation fails,
  unless the connection itself is unavailable.
- **FR-015**: The application MUST NOT require installation of a project-specific daemon or
  agent on a managed host for the MVP.

### Key Entities

- **Connection Profile**: A named set of information used to reach one Podman host, including
  its display name, location, and authentication reference.
- **Podman Host**: The local or remote environment that exposes container state and operations.
- **Container**: A workload managed by a Podman host, with identity, image, lifecycle state,
  ports, mounts, networks, logs, and resource activity.
- **Operation**: A requested read or lifecycle action, with target, start time, progress state,
  result, and error details when applicable.
- **Live Stream**: A time-ordered sequence of log or metric updates associated with one container
  and one connection.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: With a valid saved connection, an operator can reach the container inventory in
  three deliberate interactions or fewer after launching the application.
- **SC-002**: On a host containing up to 100 containers, the inventory displays the first usable
  result within 2 seconds after a successful response begins.
- **SC-003**: 100% of stop, restart, and remove actions shown in acceptance testing identify the
  exact target and require explicit confirmation before the host is changed.
- **SC-004**: In acceptance testing, an operator can inspect a container and complete a normal
  lifecycle action without leaving the application in at least 9 out of 10 attempts.
- **SC-005**: A running log view displays new output within 2 seconds of its availability under
  normal local-network conditions.
- **SC-006**: Release artifacts can be produced for macOS, Linux, and Windows from the same
  repository, and the application starts on each platform without an additional runtime.
- **SC-007**: The default automated test suite completes without a live Podman host and reports
  connection, rendering, safety-confirmation, and error-state behavior.

## Assumptions

- The operator has a Podman host that is already installed, running, and reachable through a
  supported local or remote connection.
- Authentication is delegated to the operating system or the configured connection mechanism;
  the application does not introduce a credential vault in the MVP.
- One host is active at a time in the MVP; saved profiles are supported, but fleet-wide
  aggregation and bulk actions are out of scope.
- Container creation, image building, registry management, volume administration, network
  administration, and pod orchestration are out of scope for the first release.
- A UTF-8 terminal with keyboard input is available on each supported platform.
