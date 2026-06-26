# Copy Resource

## Purpose

The copy resource ensures a target file or directory is a physical copy of a
source file or directory.

Use it when a tool needs real files on disk rather than symlinks. For example,
Codex skills should be materialized as real directories:

```txt
~/.codex/skills/nuxt-practices/SKILL.md
```

## Config

```yaml
copies:
  - source: ./codex/skills/nuxt-practices
    target: ~/.codex/skills/nuxt-practices
    replace: false
```

`source` and `target` can be files or directories. Relative paths resolve from
the directory containing the selected config file.

## Status check

Satisfied when:

- source exists
- target exists
- target has the same file and directory contents as source

Missing when target does not exist.

Changed when target exists and differs from source and `replace: true`.

Failed when:

- source is missing
- source is a symlink
- source contains a symlink
- target has a symlinked ancestor
- target differs and `replace: false`

## Apply

If target is missing, Kitout copies the source to the target.

If target exists and differs, Kitout only replaces it when `replace: true`.
Replacement removes the existing target before copying the source again.

## Safety

Default to `replace: false`.

Do not overwrite a real file or directory unless the config explicitly allows
it. `kitout apply` asks for confirmation before replacing an existing copy
target unless `--yes` is passed. Creating a missing copy target does not require
confirmation.

Copy sources must not be symlinks and must not contain symlinks. This keeps the
result physically materialized and avoids surprising discovery behavior in tools
that do not follow symlinked directories reliably.

Copy targets also reject symlinked ancestors before status planning or apply.
This keeps missing-target creation from writing through a linked parent into a
different location than the configured target path.

## Implementation status

Implemented as `resources.CopyResource`. Status compares regular file contents
and directory trees recursively. Apply copies regular files and directories, and
dry-run plans never write files.

## Shared expectations

Every resource must support:

- status check
- apply
- dry-run plan
- readable result messages
- unit tests

Status must never change the system.

Apply should be idempotent.
