# Directory Resource

## Purpose

The directory resource ensures a directory exists.

## Config

```yaml
directories:
  - ~/code
  - ~/.config
```

Future expanded form:

```yaml
directories:
  - path: ~/code
    mode: "0755"
```

## Status check

Satisfied when the path exists and is a directory.

Missing when the path does not exist.

Changed when the path exists but is not a directory.

## Apply

Create the directory and parent directories when missing.

Equivalent behavior:

```sh
mkdir -p ~/code
```

## Safety

If the path exists and is a file, do not replace it.

Report a clear error.

## Shared expectations

Every resource must support:

- status check
- apply
- dry-run plan
- readable result messages
- unit tests

Status must never change the system.

Apply must be idempotent.
