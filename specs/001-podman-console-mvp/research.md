# Research: Podman Console MVP

## Decision 1: Use the Podman Go bindings

**Decision**: Depend on the versioned `go.podman.io/podman/v6/pkg/bindings` module and wrap it
behind an internal `PodmanClient` interface.

**Rationale**: Podman's official bindings are intended for Go applications, expose container
listing, inspection, lifecycle operations, and streaming operations, and can connect to a local
or remote Podman service. This keeps Podman semantics in the Podman project instead of copying
them into this application.

**Sources**:

- [Podman Go bindings README](https://github.com/podman-container-tools/podman/blob/main/pkg/bindings/README.md)
- [Podman remote client tutorial](https://github.com/podman-container-tools/podman/blob/main/docs/tutorials/remote_client.md)
- [Podman service destination examples](https://github.com/containers/common/blob/main/docs/containers.conf.5.md)

**Alternatives considered**:

- Calling the `podman` executable and parsing text: rejected because it requires a separate
  client installation and makes the UI dependent on human-formatted output.
- Implementing the REST protocol directly: rejected for the MVP because it duplicates Podman
  transport, streaming, and version compatibility logic.
- Importing the entire Podman CLI: rejected because it adds a larger surface and couples the
  application to command registration rather than the service contract.

## Decision 2: Use Bubble Tea v2 for the TUI

**Decision**: Use `charm.land/bubbletea/v2`, with Bubbles v2 components and Lip Gloss v2 for
layout and styling.

**Rationale**: The current framework provides an event-driven model/update/view architecture,
keyboard handling, alternate-screen support, and portable terminal rendering. Bubbles supplies
the table, viewport, help, and input primitives needed by the MVP, while keeping the application
logic in our own model.

**Sources**:

- [Bubble Tea README](https://github.com/charmbracelet/bubbletea)
- [Bubbles table component](https://github.com/charmbracelet/bubbles/blob/main/table/table.go)
- [Lip Gloss repository](https://github.com/charmbracelet/lipgloss)

**Alternatives considered**:

- `tview`/`tcell`: proven and used by the official Podman TUI, but the Bubble Tea v2 model is a
  better fit for cancellable async commands and a deliberately modern component layout.
- Rust/Ratatui: technically suitable, but Go reduces friction with Podman bindings and matches
  Gabin's existing systems tooling.
- A web UI: rejected because the product goal is a terminal-first operator workflow with no
  browser or service to deploy.

## Decision 3: Persist only connection metadata

**Decision**: Store a JSON file below the platform user configuration directory using the standard
  library's `os.UserConfigDir`. A profile contains a display name, service URI, and optional
  identity-file reference. The file never contains private key material, passwords, or tokens.

**Rationale**: JSON is sufficient for a small local configuration and avoids adding a parser or
  embedded database. The platform API gives correct locations on macOS, Linux, and Windows.

**Alternatives considered**:

- SQLite: rejected as unnecessary for a handful of profiles.
- YAML/TOML: rejected to keep the dependency and configuration surface smaller.
- Reusing Podman's full configuration parser: deferred; importing existing Podman connections can
  be added after the explicit profile workflow is reliable.

## Decision 4: Make the UI asynchronous at the boundary

**Decision**: Every host request runs through a cancellable `context.Context` and returns a typed
  Bubble Tea message. The UI never calls a binding method directly during rendering or key-event
  handling.

**Rationale**: Remote calls and log streams can block. A single event-loop boundary prevents
  stale responses from corrupting the current screen and lets the UI display progress, preserve
  partial logs, and cancel work.

**Alternatives considered**:

- Synchronous requests in `Update`: rejected because one slow host would freeze navigation and
  violate the operator-first principle.
- A general worker pool: deferred until measured need; one command per user request is simpler for
  the MVP.

## Decision 5: Test with a fake host by default

**Decision**: The app-level port is tested with a deterministic fake that can return inventories,
  details, operations, log chunks, metrics, and typed failures. Live Podman tests are opt-in.

**Rationale**: The default test suite must be fast, repeatable, and runnable on all three client
  platforms even when no Podman machine is installed.

**Alternatives considered**:

- Requiring a Podman machine in CI: rejected because it makes Windows and macOS runners more
  fragile and does not improve pure UI/domain coverage.
- Mocking individual binding functions: rejected because it couples tests to a third-party
  implementation rather than our own contract.
