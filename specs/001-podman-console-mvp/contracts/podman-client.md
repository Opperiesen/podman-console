# Podman Client Contract

The application layer depends on this domain-facing contract. The binding adapter is the only
package allowed to translate Podman binding types.

## Connection

```text
Connect(ctx, ConnectionProfile) -> Session or typed error
Close() -> error
```

The session is scoped to one selected profile. A connection error must classify whether the
failure is invalid configuration, authentication/authorization, transport, or host service.

## Inventory and detail

```text
ListContainers(ctx, all=true) -> []ContainerSummary or typed error
InspectContainer(ctx, containerID) -> ContainerDetails or typed error
```

`ListContainers` returns an empty slice for a reachable host with no containers. It must not
convert a timeout or connection error into an empty result.

## Lifecycle

```text
Start(ctx, containerID) -> error
Stop(ctx, containerID) -> error
Restart(ctx, containerID) -> error
Remove(ctx, containerID) -> error
```

The adapter does not perform retries for mutations. The caller refreshes the inventory after a
successful mutation and preserves the original host error after a failure.

## Streams

```text
Logs(ctx, containerID, options) -> receive LogLine values until EOF or error
Stats(ctx, containerID, interval) -> receive ContainerStats values until cancellation or error
```

Streams must honor context cancellation. The caller owns the channel or iterator lifecycle and
must be able to preserve already received values when the stream terminates with an error.

## Error contract

Errors expose:

- a stable category for UI treatment;
- the original error for logging and troubleshooting;
- the target identifier when the failure concerns a container.

The adapter must never silently discard a non-empty host error body.
