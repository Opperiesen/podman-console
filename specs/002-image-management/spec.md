# Feature Specification: Podman Image Management

**Feature Branch**: `002-image-management`

**Created**: 2026-08-25

**Status**: Draft

**Input**: Post-MVP follow-up for the public `v0.1.0` release of Podman Console.

## User Scenarios & Testing

### User Story 1 - Browse local images (Priority: P1)

As an operator connected to a Podman host, I want to see the images stored on that host so that I
can understand what is available before starting or troubleshooting containers.

**Why this priority**: An image inventory is the smallest useful extension beyond container
management and provides the foundation for every later image operation.

**Independent Test**: Connect to a fake host containing tagged, untagged, and dangling images,
open the Images view, filter the inventory, and inspect an image without changing the host.

**Acceptance Scenarios**:

1. **Given** an active host with images, **when** the operator opens the Images view, **then** the
   view shows repository/tag, short ID, digest when available, size, creation time, and image
   state for every returned image.
2. **Given** an active host with no images, **when** the operator opens the Images view, **then**
   the interface shows an explicit empty state and keeps refresh and pull actions available.
3. **Given** an image inventory, **when** the operator enters a filter, **then** matching
   repository, tag, ID, or digest values remain visible and non-matching rows are hidden without
   another host request.
4. **Given** a selected image, **when** the operator opens its details, **then** the interface
   shows the full ID, all repository tags and digests, size, creation time, labels when present,
   and the number of containers using it.

### User Story 2 - Pull an image with visible progress (Priority: P2)

As an operator, I want to pull an image by reference and see ordered progress so that I can make a
new image available without switching to another terminal.

**Why this priority**: Pulling is the principal safe way to add an image and turns the inventory
into an actionable workflow while keeping registry credentials delegated to Podman.

**Independent Test**: Start from a fake host without the requested image, enter a valid image
reference, feed progress and completion events, and verify the result and authoritative refresh.

**Acceptance Scenarios**:

1. **Given** an active host, **when** the operator enters a non-empty registry image reference and
   confirms the pull, **then** the interface starts one pull request and displays progress in
   arrival order with the target reference and host visible.
2. **Given** a successful pull, **when** the Podman stream completes, **then** the interface shows
   success and refreshes the image inventory from the host rather than adding a guessed row.
3. **Given** a pull that reports an error or disconnects, **when** the stream ends, **then** the
   interface preserves the received progress, shows an actionable error, and does not claim that
   the image is available.
4. **Given** a pull in progress, **when** the operator cancels it, **then** the request context is
   cancelled, the interface reports that live progress stopped, and a later refresh determines
   the authoritative image state.

### User Story 3 - Remove one image safely (Priority: P3)

As an operator, I want to remove one selected image with an exact-target confirmation so that I can
free storage without accidentally pruning unrelated images or images used by a container.

**Why this priority**: Removal is valuable but destructive, so it follows the read and pull flows
and must reuse the existing target-aware safety contract.

**Independent Test**: Select one fake image, cancel its confirmation once, confirm it once, and
verify that only the confirmed image request is sent and the inventory refreshes afterward.

**Acceptance Scenarios**:

1. **Given** a selected image, **when** the operator requests removal, **then** the confirmation
   names the active host, image reference, and full or unambiguous image ID and does not enable a
   bulk or force-removal path.
2. **Given** a pending removal confirmation, **when** the operator cancels, **then** no mutation
   request is sent and the image remains selected.
3. **Given** a confirmed image removal, **when** Podman accepts it, **then** the interface reports
   the outcome and refreshes the inventory from the host.
4. **Given** an image used by a container or an unauthorized target, **when** removal fails,
   **then** the interface keeps the inventory intact and explains the host error without retrying
   a force removal.

### Edge Cases

- An image can have several repository tags, no tag, or no digest; the view must display explicit
  empty values instead of shifting columns or inventing names.
- A pull reference can be syntactically non-empty but rejected by Podman or the registry; the
  error must be shown without persisting the reference as an image.
- Registry authentication, certificates, signature policy, and remote credential handling remain
  owned by the configured Podman service; Podman Console must not collect or persist passwords.
- A stale selected image or changed active host must invalidate a pending inspect, pull result, or
  removal confirmation before it can mutate the wrong target.
- Pull progress may contain partial lines, repeated layer updates, an unexpected JSON message, or
  a clean EOF; the stream view must remain ordered and terminate once the request completes.
- An image may be removed by another client between selection and confirmation; the resulting
  not-found response must be actionable and must not be treated as successful deletion.
- An image inventory larger than the terminal height must remain navigable without truncating the
  selected row or hiding the active target.

## Requirements

### Functional Requirements

- **FR-001**: The system MUST provide an Images view for the currently active Podman connection.
- **FR-002**: The system MUST display image repository/tag values, short ID, digest when
  available, size, creation time, and explicit empty values for missing fields.
- **FR-003**: The system MUST allow local filtering of the image inventory by repository, tag,
  ID, or digest without issuing a new host request.
- **FR-004**: The system MUST provide image details containing the full ID, all tags and digests,
  size, creation time, labels when present, and the number of associated containers.
- **FR-005**: The system MUST validate that a pull reference is non-empty before sending a pull
  request and MUST preserve the active host identity throughout the operation.
- **FR-006**: The system MUST display pull progress in arrival order and preserve partial progress
  when a pull fails, is cancelled, or disconnects.
- **FR-007**: The system MUST refresh the image inventory authoritatively after a successful pull
  or removal instead of synthesizing a local result.
- **FR-008**: The system MUST require target-aware confirmation before removing an image and MUST
  send no mutation request when the operator cancels.
- **FR-009**: The system MUST never force-remove an image or remove all images as part of this
  feature.
- **FR-010**: The system MUST classify transport, authorization, registry, not-found, in-use,
  cancellation, and malformed-stream failures into actionable interface feedback.
- **FR-011**: The system MUST delegate registry credentials and trust policy to the configured
  Podman service and MUST NOT store passwords or private keys in Podman Console configuration.
- **FR-012**: The default automated test suite MUST exercise image list, inspect, pull, stream,
  cancellation, removal, stale-target, and error behavior without requiring a live Podman host.
- **FR-013**: Live integration validation MUST be opt-in, use an explicitly disposable image
  reference, and leave the test host with no image created solely by the test.

### Key Entities

- **ImageSummary**: A locally stored image row with full ID, display references, digest, size,
  creation time, dangling/read-only state, and container usage count.
- **ImageDetails**: The authoritative inspect result for one image, including all tags, digests,
  labels, size, creation time, and identity fields.
- **ImagePullEvent**: An ordered progress, completion, cancellation, or error observation tied to
  one image reference and one active connection.
- **ImageOperation**: The target-aware state of a pull or removal, including operation kind,
  target identity, status, partial output, and final feedback.

## Success Criteria

### Measurable Outcomes

- **SC-001**: On a host with up to 100 local images, the first usable inventory result is shown
  within 2 seconds after a successful response begins.
- **SC-002**: An operator can identify an image by repository/tag, ID, or digest and open its
  details in three deliberate interactions or fewer from the Images view.
- **SC-003**: 100% of image-removal acceptance tests identify the exact host and image and require
  explicit confirmation before the host is changed.
- **SC-004**: During an accepted pull, the first progress feedback appears within 2 seconds of
  its availability and remains ordered until completion or cancellation.
- **SC-005**: The default test suite covers successful and failed image list, inspect, pull, and
  removal flows without Podman installed or reachable.
- **SC-006**: The image feature preserves the existing Darwin, Linux, and Windows amd64/arm64
  build matrix and the existing container workflows.

## Assumptions and Scope Boundaries

- The operator already has a configured local or remote Podman service; one host remains active at
  a time.
- Podman’s existing registry authentication, certificate, and signature-policy configuration is
  authoritative. No login, password prompt, or credential vault is added.
- Pull accepts one image reference at a time and uses the host’s default pull policy; architecture,
  platform overrides, and all-tags pulls are out of scope for this feature.
- Removal is one image at a time and never uses force, prune, or all-images behavior.
- Image build, push, search, save/load, signing, registry administration, volume management,
  pod management, bulk operations, and multi-host aggregation are out of scope.
- The feature extends the existing `PodmanClient` port and TUI state model without adding a new
  daemon, database, or runtime dependency.
