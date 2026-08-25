# Data Model: Podman Image Management

## ImageSummary

Represents one row in the image inventory.

| Field | Meaning |
|---|---|
| `ID` | Full image identifier; the UI may render an unambiguous short form |
| `References` | Repository tags and other displayable names |
| `Digests` | Repository digests when supplied by the host |
| `Size` | Local image size in bytes |
| `CreatedAt` | Host-reported creation time |
| `Containers` | Number of containers associated with the image |
| `Dangling` | Whether the host reports the image as dangling |
| `ReadOnly` | Whether the host reports the image as read-only |

## ImageDetails

Authoritative inspect data for one image. It includes the full identity, all references and
digests, size, creation time, labels when present, and the host’s container usage information. The
model retains the requested image identity so a late response cannot replace details for a newer
selection.

## ImagePullEvent

An ordered observation for one pull operation.

| Field | Meaning |
|---|---|
| `Reference` | Image reference requested by the operator |
| `Kind` | Progress, success, error, cancelled, or ended |
| `Text` | Human-readable stream fragment or actionable error |
| `ImageIDs` | IDs reported by a successful host response |
| `Target` | Active connection identity captured at operation start |

## ImageOperation

Model state for a pull or removal. It records the operation kind, exact target identity, current
status, preserved progress or feedback, and cancellation state. An operation can update the UI but
cannot authorize a mutation after the active connection or selected image changes.

## State transitions

```text
idle -> requested -> running -> succeeded -> refreshing -> idle
                         \-> failed --------^
                         \-> cancelled -----^
```

Removal uses `requested -> confirming -> running -> succeeded/failed -> refreshing -> idle` and
never enters a force or bulk state.
