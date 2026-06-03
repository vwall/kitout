# Shell Command Resource

## Purpose

The shell command resource handles setup steps that do not yet deserve a first-class resource.

## Config

```yaml
shell:
  - name: Enable Corepack
    command: corepack enable
    when: missing-command:pnpm
```

## Fields

```txt
name       required human-readable label
command    required command to run
when       optional condition
```

## MVP conditions

Start with a small set of conditions:

```txt
always
missing-command:<name>
exists:<path>
missing:<path>
```

## Status check

If condition is satisfied and no action is needed, mark satisfied.

If condition indicates the command should run, mark missing.

If condition cannot be evaluated, mark failed.

## Apply

Run the configured command through the user's shell.

## Safety

Shell commands are powerful and risky.

Rules:

- never run shell commands that are not explicitly listed
- show the command in dry-run
- require confirmation unless `--yes` is used
- capture exit code and output
- do not hide failures

## Future direction

Over time, repeated shell commands should become first-class resources.

## Implementation status

Implemented as `resources.ShellCommandResource`. The MVP supports `always`,
`missing-command:<name>`, `exists:<path>`, and `missing:<path>` conditions.
Configured commands run through `sh -c` using the shared command runner
interface. `kitout apply` requires confirmation before running a shell command
unless `--yes` is passed.

## Shared expectations

Every resource must support:

- status check
- apply
- dry-run plan
- readable result messages
- unit tests

Status must never change the system.

Apply must be idempotent.
