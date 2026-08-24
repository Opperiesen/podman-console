<!--
Sync Impact Report
- Version change: template → 1.0.0
- Modified principles: none; established the initial project principles
- Added sections: Product Constraints, Development Workflow
- Removed sections: none
- Follow-up TODOs: none
-->

# Podman Console Constitution

## Core Principles

### I. Operator-First Experience

The application MUST reduce the time and cognitive load required to inspect and operate
Podman. Every screen MUST expose the current target, state, and available actions clearly.
The interface MUST remain usable from a keyboard and a standard terminal without requiring a
web browser or a background control plane.

### II. Native Podman Integration

The application MUST use documented Podman interfaces as its source of truth. It MUST NOT
reimplement container state management or infer state by scraping human-formatted command
output. Local and remote connections MUST share the same domain model and error handling.

### III. Safe-by-Default Operations

Read-only inspection MUST work without confirmation. Actions that can stop, remove, recreate,
or otherwise disrupt workloads MUST identify the exact target and require an explicit
confirmation. The application MUST surface the operation and the returned error when an
action fails; it MUST NOT hide partial results.

### IV. Cross-Platform, Single-Binary Delivery

The client MUST build and run on macOS, Linux, and Windows from the same codebase. The
application MUST NOT require a project-specific daemon or agent on a managed Podman host for
the MVP. Platform-specific behavior MUST be isolated behind small interfaces and covered by
tests where it affects user-visible behavior.

### V. Small, Tested Increments

Every user-visible capability MUST have a runnable acceptance scenario and focused automated
tests for its domain behavior. New dependencies MUST have a concrete purpose and a maintained
cross-platform story. The project MUST prefer a small vertical slice that can be released and
used over speculative abstractions.

## Product Constraints

The initial product is a terminal UI for administering Podman hosts. The MVP covers connection
selection, container listing, detail inspection, lifecycle actions, logs, and basic resource
metrics. Orchestration, image building, registry management, and fleet-wide automation remain
out of scope until the single-host workflow is reliable.

The implementation will use Go for its portable distribution and alignment with the Podman
ecosystem. The UI toolkit and transport libraries are implementation details, but each must
support macOS, Linux, and Windows without a mandatory runtime installation.

## Development Workflow

Changes MUST be organized around a user-visible slice or a clearly justified foundational
change. Formatting, static analysis, unit tests, and cross-platform build checks MUST pass before
merging. External Podman integration tests MUST be deterministic and opt-in; the default test
suite MUST run without a live container host.

The repository MUST document keyboard bindings, supported Podman versions, connection setup,
and the safety behavior of destructive actions. Breaking changes to the connection model or
public configuration require a release note and migration guidance.

## Governance
<!-- Example: Constitution supersedes all other practices; Amendments require documentation, approval, migration plan -->

This constitution is the highest-level project guidance. Amendments MUST update the Sync Impact
Report, increment the semantic version, and explain their effect on existing behavior. Every
implementation plan MUST include a constitution check; any violation MUST be explicit and
justified. Reviews MUST verify the safety, platform, and testing principles above.

**Version**: 1.0.0 | **Ratified**: 2026-08-24 | **Last Amended**: 2026-08-24
<!-- Example: Version: 2.1.1 | Ratified: 2025-06-13 | Last Amended: 2025-07-16 -->
