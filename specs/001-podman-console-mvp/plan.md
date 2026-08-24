# Implementation Plan: Podman Console MVP

**Branch**: `001-podman-console-mvp` | **Date**: 2026-08-24 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/001-podman-console-mvp/spec.md`

## Summary

Podman Console is an independent, cross-platform terminal application for inspecting and
operating one Podman host at a time. The MVP will present a connection selector, a responsive
container inventory, a detail view, explicit lifecycle confirmations, logs, and basic resource
metrics. The application will use Podman's Go bindings through a small internal port so the UI
does not know about transport details and the default test suite can use a deterministic fake.

## Technical Context

**Language/Version**: Go 1.25.9 minimum; development toolchain Go 1.26.x

**Primary Dependencies**:

- `go.podman.io/podman/v6/pkg/bindings` for documented Podman service access
- `charm.land/bubbletea/v2` for the event-driven TUI runtime
- `charm.land/bubbles/v2` for table, viewport, help, and input components
- `charm.land/lipgloss/v2` for layout and styling
- Go standard library for configuration, context cancellation, JSON, and testing

**Storage**: A small JSON configuration file under the platform user config directory. It stores
connection names, Podman service URIs, and authentication references such as identity-file paths;
it does not store private keys or passwords.

**Testing**: Go `testing`, deterministic fake Podman client, Bubble Tea model tests, and
cross-platform `go build` checks. Live-host integration tests are opt-in and are not required for
the default suite.

**Target Platform**: macOS, Linux, and Windows terminals; local or remote Podman service.

**Project Type**: Cross-platform desktop CLI/TUI application.

**Performance Goals**: Keep the event loop responsive while requests run; render the first
inventory rows within 2 seconds after a successful host response for up to 100 containers; show
new followed log output within 2 seconds under normal local-network conditions.

**Constraints**: No project-specific daemon or host agent; no credentials in application config;
destructive actions require explicit target-aware confirmation; default tests must not require a
live Podman host; terminal minimum of 100 columns by 24 rows for the full layout.

**Scale/Scope**: One active host, saved connection profiles, up to 100 visible containers in the
MVP; no fleet aggregation, bulk actions, image building, registry administration, or pod
orchestration.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Check | Result |
|---|---|---|
| Operator-First Experience | Inventory, detail, logs, metrics, visible target, keyboard hints | PASS |
| Native Podman Integration | Official Go bindings behind a `PodmanClient` port; no human-output parsing | PASS |
| Safe-by-Default Operations | Stop, restart, and remove use target-aware confirmation; reads stay available | PASS |
| Cross-Platform, Single-Binary Delivery | Go build targets macOS, Linux, and Windows; no host agent | PASS |
| Small, Tested Increments | Each user story has a fake-backed acceptance path and unit coverage | PASS |

No constitution violations require a complexity exception.

## Project Structure

### Documentation (this feature)

```text
specs/001-podman-console-mvp/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── podman-client.md
│   └── keyboard-and-safety.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/podman-console/
└── main.go

internal/
├── app/
│   ├── app.go
│   ├── messages.go
│   └── model.go
├── config/
│   ├── model.go
│   └── store.go
├── domain/
│   ├── connection.go
│   ├── container.go
│   └── operation.go
├── podman/
│   ├── client.go
│   ├── bindings.go
│   └── errors.go
└── ui/
    ├── components.go
    ├── keys.go
    ├── layout.go
    └── styles.go

tests/
├── fixtures/
└── integration/

docs/
├── connections.md
└── keybindings.md

.github/workflows/
└── ci.yml
```

**Structure Decision**: Use a single Go module with an executable under `cmd/` and private
packages under `internal/`. Domain types remain independent from the Podman binding types. The
`internal/podman` adapter translates official binding responses into domain values, while
`internal/app` owns asynchronous commands and screen state. Tests colocated with packages cover
domain and UI behavior; `tests/integration` is reserved for opt-in live-host scenarios.

## Phase 0: Research Decisions

The research record is in [research.md](./research.md). Key decisions:

1. Use Podman's official Go bindings rather than shelling out to the CLI or parsing human output.
2. Use the Podman service URI model for local Unix sockets and remote SSH connections.
3. Use Bubble Tea v2 and its current Bubbles/Lip Gloss modules for a modern, portable TUI.
4. Keep connection persistence deliberately small and credential-free.

## Phase 1: Design Summary

The data model is in [data-model.md](./data-model.md). The binding-facing port and UI safety
contract are in [contracts/](./contracts/). Runnable validation scenarios are in
[quickstart.md](./quickstart.md).

The first vertical slice is User Story 1: a fake host can be selected, queried asynchronously,
and rendered in the inventory. User Stories 2 and 3 then reuse that port and selection state for
mutations, logs, and metrics.

## Constitution Check — Post-Design

| Principle | Post-design evidence | Result |
|---|---|---|
| Operator-First Experience | Inventory/detail screens and responsive async commands are isolated from transport | PASS |
| Native Podman Integration | Adapter calls bindings and exposes only domain values to the UI | PASS |
| Safe-by-Default Operations | Safety contract requires exact target confirmation before mutation | PASS |
| Cross-Platform, Single-Binary Delivery | Platform-neutral config path and binding URI; CI builds three OS families | PASS |
| Small, Tested Increments | Fake client, model tests, and quickstart scenarios cover each story | PASS |

No violations introduced during design.

## Complexity Tracking

No exceptions.
