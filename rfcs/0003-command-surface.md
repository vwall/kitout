# RFC 0003: Command Surface

## Status

Draft

## Context

Kitout should feel simple and obvious. The first commands should cover initialization, checking, applying, and diagnosing setup problems.

## Decision

The MVP command surface is:

```sh
kitout init
kitout status
kitout apply
kitout apply --dry-run
kitout doctor
kitout version
```

Optional:

```sh
kitout list
```

## Consequences

Benefits:

- small command surface
- easy to explain
- maps to Bork-like status and satisfy flows without copying names
- gives users a safe dry-run path

Costs:

- no advanced workflows at first
- no automatic config editing beyond init

## Alternatives considered

### `kitout satisfy`

Rejected for the initial command surface because `apply` is more common and clearer.

### `kitout reconcile`

Accurate, but too formal for the primary command.

### `kitout bootstrap`

Useful for first install, but less natural for repeated runs.
