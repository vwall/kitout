# RFC 0001: Core Resource Model

## Status

Draft

## Context

Kitout needs a consistent way to represent desired machine state. The tool will manage different resource types, but each resource should follow the same lifecycle.

## Decision

Represent each desired state item as a resource with a stable ID, type, status check, and apply operation.

```go
type Resource interface {
    ID() string
    Type() string
    Status(ctx context.Context) (StatusResult, error)
    Apply(ctx context.Context) (ApplyResult, error)
}
```

The engine will not know the details of each resource. It will only orchestrate resource checks and applies.

## Consequences

Benefits:

- simple mental model
- easy to test resources independently
- easy to add resources later
- consistent output

Costs:

- some resource types may need extra metadata
- dependency handling is limited at first

## Alternatives considered

### Imperative scripts

Rejected for the core model because scripts are hard to inspect safely and hard to make idempotent.

### Full dependency graph

Deferred. A fixed execution order is enough for the MVP.

### Plugin API

Rejected for the MVP. Built-in resources are easier to test and document.
