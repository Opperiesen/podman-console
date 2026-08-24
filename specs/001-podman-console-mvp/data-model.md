# Data Model: Podman Console MVP

## ConnectionProfile

Represents one named route to a Podman service.

| Field | Type | Rules |
|---|---|---|
| `name` | string | Required, unique within the config, trimmed, 1–64 characters |
| `uri` | string | Required, parsed as a supported Podman service URI |
| `identity_path` | string | Optional path reference; must not be persisted as key contents |
| `default` | boolean | At most one profile is default |

The profile is local metadata. Credentials and private key contents are owned by the operating
system or the configured SSH mechanism.

## ContainerSummary

Represents one row in the inventory.

| Field | Type | Rules |
|---|---|---|
| `id` | string | Required, stable host identifier |
| `name` | string | Required; generated names are valid |
| `image` | string | Required when supplied by the host; otherwise display an explicit unknown value |
| `state` | enum | `created`, `running`, `paused`, `stopped`, `exited`, `unknown` |
| `status` | string | Host-provided human-readable status, never used for state decisions |
| `ports` | list | May be empty |

## ContainerDetails

Extends `ContainerSummary` with immutable or slowly changing metadata.

| Field | Type | Rules |
|---|---|---|
| `created_at` | timestamp | Optional when host does not provide it |
| `command` | list of strings | May be empty; rendered without shell re-execution |
| `mounts` | list of Mount | May be empty |
| `networks` | list of NetworkAttachment | May be empty |
| `labels` | map of strings | May be empty |

`Environment` is intentionally omitted from the first detail view to avoid casually exposing
secrets in a shared terminal or screenshot.

## ContainerStats

Represents one point in a resource stream.

| Field | Type | Rules |
|---|---|---|
| `container_id` | string | Required and must match the active selection |
| `cpu_percent` | number | Optional; display unavailable when absent |
| `memory_usage_bytes` | integer | Optional, non-negative |
| `memory_limit_bytes` | integer | Optional, non-negative |
| `memory_percent` | number | Optional |
| `observed_at` | timestamp | Required for a displayed sample |

## Operation

Tracks one requested action.

| Field | Type | Rules |
|---|---|---|
| `kind` | enum | `list`, `inspect`, `start`, `stop`, `restart`, `remove`, `logs`, `stats` |
| `target_id` | string | Required for container actions |
| `target_name` | string | Captured when confirmation is requested |
| `status` | enum | `pending`, `succeeded`, `failed`, `cancelled` |
| `error` | string | Present only for failed operations; preserve host detail |

## LiveStream

Represents an active or completed log/metric subscription.

| Field | Type | Rules |
|---|---|---|
| `kind` | enum | `logs` or `stats` |
| `container_id` | string | Required |
| `items` | ordered list | Preserve arrival order and already received items |
| `current` | boolean | False after cancellation, disconnect, or target removal |
| `error` | string | Optional explanation for a stopped stream |

## State transitions

```text
pending -> succeeded
pending -> failed
pending -> cancelled

running LiveStream -> current=false on cancel, disconnect, or target disappearance
```

The UI treats host state as authoritative after every mutation and refreshes the selected
container instead of applying a locally guessed state transition.
