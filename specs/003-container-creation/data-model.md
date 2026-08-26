# Data Model: Run a Container from a Local Image

## ContainerCreateRequest

The immutable mutation input captured when the operator confirms the form.

| Field | Meaning |
|---|---|
| `ImageID` | Full ID of the local image selected from the authoritative inventory |
| `ImageReference` | Display reference captured for confirmation and feedback |
| `Name` | Explicit validated container name |
| `Command` | Optional ordered argument list; empty means image default command |

The request carries no registry credentials, environment map, mount, network, privilege, resource,
restart, or attach settings.

## ContainerRunResult

The result of the ordered create/start sequence.

| Field | Meaning |
|---|---|
| `ContainerID` | Exact ID returned by Podman create, when creation succeeded |
| `Started` | Whether Podman accepted the subsequent start request |
| `Warnings` | Non-fatal warnings returned by the create response |

`ContainerID` may be non-empty while `Started` is false. That state is a partial outcome, not a
successful run.

## ContainerCreateOperation

Application state associated with one form submission.

| Field | Meaning |
|---|---|
| `Target` | Active connection identity captured at opening |
| `ImageGeneration` | Inventory generation captured with the selected image |
| `Request` | Immutable create request after validation |
| `Status` | Editing, confirming, creating, starting, succeeded, partial, failed, or cancelled |
| `Result` | Optional `ContainerRunResult` |
| `Error` | Typed actionable error, when present |

## State transitions

```text
idle -> editing -> confirming -> creating -> starting -> refreshing -> succeeded
                    │             │          ├─ failed
                    │             │          └─ cancelled/partial -> refreshing
                    │             └─ failed
                    └─ cancelled
```

The operation generation and active connection identity gate every asynchronous message. A late
message cannot change another image selection, connection, or form.
