# Symlink Resource

## Purpose

The symlink resource ensures a target path points to a source path.

## Config

```yaml
symlinks:
  - source: ~/dotfiles/home/zshrc
    target: ~/.zshrc
    replace: false
```

## Status check

Satisfied when:

- target exists
- target is a symlink
- target points to source

Missing when target does not exist.

Changed when target exists but points somewhere else.

Failed when target exists as a normal file or directory and replacement is not allowed.

## Apply

If target is missing:

```sh
ln -s <source> <target>
```

If target exists and `replace: true`, replace it safely.

If target exists and `replace: false`, fail with guidance.

## Safety

Default to `replace: false`.

Do not overwrite a real file unless the config explicitly allows it.

`kitout apply` also asks for confirmation before replacing an existing symlink
target unless `--yes` is passed. Creating a missing symlink does not require
confirmation.

A future backup option may support:

```yaml
backup: true
```

## Implementation status

Implemented as `resources.SymlinkResource`. Status uses `os.Lstat` so it can
inspect links without following them, and apply only replaces an existing target
when `replace: true` is configured. CLI apply confirmation is required for
replacement unless `--yes` is passed.

## Shared expectations

Every resource must support:

- status check
- apply
- dry-run plan
- readable result messages
- unit tests

Status must never change the system.

Apply must be idempotent.
