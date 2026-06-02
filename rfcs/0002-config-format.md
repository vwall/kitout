# RFC 0002: Config Format

## Status

Draft

## Context

Kitout needs a human-editable config format that is easy to read, easy to generate, and easy for Codex to modify.

## Decision

Use YAML for the MVP config format.

Require:

```yaml
version: 1
```

Support top-level sections:

```yaml
brew:
casks:
directories:
repos:
symlinks:
macos_defaults:
shell:
```

## Consequences

Benefits:

- readable by humans
- common for developer tools
- easy to document
- supports simple and expanded forms

Costs:

- YAML has edge cases
- comments are not preserved if the tool rewrites config
- schema validation must be strict

## Alternatives considered

### TOML

TOML is simpler to parse in some ways, but nested repeated resources can become verbose.

### JSON

JSON is easy for machines, but not pleasant for hand-written setup files.

### Bash DSL

Rejected because Kitout should not parse shell syntax or reproduce Bork's implementation model.
