# RFC Process

## Purpose

RFCs record meaningful design decisions before implementation gets too far ahead of the product.

## When to write an RFC

Write an RFC for decisions that affect:

- config schema
- resource model
- command behavior
- safety rules
- public output format
- distribution strategy
- compatibility promises

Do not write RFCs for small implementation details.

## RFC status values

```txt
Draft
Accepted
Rejected
Superseded
Implemented
```

## Current RFCs

- `0001-core-resource-model.md`
- `0002-config-format.md`
- `0003-command-surface.md`
- `0004-asdf-resource.md`

## Template

```md
# RFC NNNN: Title

## Status

Draft

## Context

What problem are we solving?

## Decision

What are we choosing?

## Consequences

What does this make easier or harder?

## Alternatives considered

What else did we consider?
```
