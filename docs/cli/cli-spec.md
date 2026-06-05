# CLI Specification

## Command name

```sh
kitout
```

## Global flags

```txt
--config PATH       Path to config file (default: ./kitout.yaml, then ~/.config/kitout/kitout.yaml)
--verbose           Stream subprocess output during apply
--quiet             Reduce output
--no-color          Disable colored output
--json              Print machine-readable JSON output
--yes               Bypass apply confirmations for shell commands and symlink replacements
```

These flags are currently parsed by the root command and by implemented
subcommands. `doctor`, `status`, and `apply` use `./kitout.yaml` from the
current working directory when it exists, then fall back to
`~/.config/kitout/kitout.yaml`; pass `--config` to override that path. `init`
still writes `~/.config/kitout/kitout.yaml` by default unless `--config` is
passed. Human output colors status markers when stdout is an interactive terminal;
redirected output remains plain text, and `--no-color` disables ANSI color
markers. Human output includes both symbols and text labels so status remains
readable without color. `status`, `apply`, and `doctor` print a short startup
line before slower checks begin so command startup does not look stuck. `apply`
prints a progress line before each resource apply starts, so long-running
commands such as Homebrew installs and upgrades have visible activity while
default subprocess output stays captured and hidden. During human `kitout apply`
runs, `--verbose` renders each subprocess command before it starts and streams
that subprocess stdout and stderr as it runs. `--verbose` does not stream output
for `--dry-run`, `--json`, or `--quiet` runs. `--json` and `--quiet` currently
affect `status`, `apply`, and `doctor`.

When running from a private setup repo that contains `kitout.yaml`, `kitout
status`, `kitout apply`, and `kitout doctor` select that repo-local file by
default and print the selected path before running checks.

## Commands

### `kitout init`

Creates a starter config file.

```sh
kitout init
kitout init --config ./kitout.yaml
```

Behavior:

- create parent config directory if missing
- refuse to overwrite existing config unless `--force` is passed
- write a starter config that validates and can be checked by `status` without
  manual edits
- keep only deterministic directory resources active by default; package, repo,
  and dotfile examples are commented until the user customizes them

### `kitout status`

Checks configured resources.

```sh
kitout status
kitout status --json
kitout status --config ./kitout.yaml
```

Current behavior:

- load and validate the selected config file
- print a startup line naming the selected config before status checks begin
- build resources from config in stable execution order
- check resource status through the engine planner
- batch Homebrew outdated checks across brew package resources during planning
- render resource details and summary counts
- return `0` when all resources are satisfied or skipped
- return `1` when changes are needed
- return `2` for config validation, parse, or unknown-field errors
- return `3` for config read failures or failed/unknown resources
- when `--json` is passed, print machine-readable plan output

Example human output:

```txt
Kitout is checking your Mac setup...
Config: /Users/example/.config/kitout/kitout.yaml

✓ satisfied brew: git                    satisfied
! changed   brew: go                     formula is outdated
! missing   cask: ghostty                missing
✓ satisfied directory: /Users/example/code satisfied
- skip      repo: /Users/example/code/app skipped by config
× fail      shell: setup                 failed: command failed

Summary: 2 satisfied, 1 missing, 1 changed, 1 failed, 1 skipped
3 resources need attention
```

Exit behavior:

```txt
0 all resources satisfied
1 status found resources needing attention
2 validation error
3 runtime error
```

### `kitout apply`

Applies needed changes.

```sh
kitout apply
kitout apply --dry-run
kitout apply --yes
kitout apply --verbose
```

Behavior:

- load and validate config
- print a startup line naming the selected config before status checks begin
- run status checks
- build plan
- stop before applying if the plan contains failed or unknown resources
- require confirmation before implemented risky actions, currently shell commands and symlink replacements, unless `--yes` is passed
- apply missing or changed resources in stable order
- show a progress line before each resource apply starts, so long-running commands such as Homebrew upgrades do not look stuck
- by default, capture subprocess output and keep the final report concise
- with `--verbose`, render each subprocess command and stream its stdout and stderr while it runs
- render summary
- return `0` when apply completed successfully
- return `2` for validation, parse, unknown-field, or flag errors
- return `3` for config read failures or pre-apply plan failures
- return `4` for partial apply failures

### `kitout apply --dry-run`

Shows intended changes without making changes.

Example output:

```txt
Kitout is planning changes only. No changes will be made.
Config: /Users/example/.config/kitout/kitout.yaml

i Would install cask ghostty
i Would replace symlink /Users/example/.zshrc

No changes made because --dry-run was used.
No shell commands will run without explicit approval.
```

### `kitout doctor`

Checks local prerequisites and common problems.

```sh
kitout doctor
```

Checks:

- supported OS
- CPU architecture
- Xcode Command Line Tools
- Homebrew installation
- Homebrew install path for the current CPU architecture
- Git installation
- shell environment
- config file validity
- write permissions for configured filesystem targets

Current output:

```txt
Kitout is checking local prerequisites...
Config: /Users/me/.config/kitout/kitout.yaml

Doctor:
ok:   macOS                      running on macOS
ok:   CPU architecture           running on Apple Silicon
ok:   Xcode Command Line Tools   /Library/Developer/CommandLineTools
ok:   Homebrew                   Homebrew 4.0.0
ok:   Homebrew path              Homebrew prefix is /opt/homebrew
ok:   Git                        git version 2.45.0
ok:   Shell environment          SHELL and PATH look usable
ok:   Config                     config is valid
ok:   Path permissions           no configured filesystem write targets

9 total, 9 ok, 0 warnings, 0 failed
```

### `kitout version`

Prints version metadata.

```sh
kitout version
```

Current output:

```txt
kitout dev
commit unknown
built unknown
```

Release and local build artifacts should set `version`, `commit`, and `built`
metadata at build time.

## Output modes

### Human output

Default output should be concise and readable.

### JSON output

JSON output should be stable enough for tests and automation.

Example:

```json
{
  "command": "status",
  "ok": true,
  "config": {
    "path": "/Users/example/.config/kitout/kitout.yaml",
    "valid": true
  },
  "plan": {
    "summary": {
      "total": 1,
      "satisfied": 0,
      "missing": 1,
      "changed": 0,
      "failed": 0,
      "skipped": 0,
      "unknown": 0,
      "to_apply": 1
    },
    "items": [
      {
        "resource_id": "directory:/Users/example/code",
        "type": "directory",
        "state": "missing",
        "action": "apply",
        "message": "directory is missing"
      }
    ]
  }
}
```

## Confirmation rules

Interactive confirmation is required for implemented risky changes unless
`--yes` is passed.

Risky changes currently include:

- replacing symlink targets
- running shell commands

Safe changes may apply without confirmation:

- installing a missing package
- creating a missing directory
- cloning a missing repo

Dry-run only renders the plan, so it does not ask for confirmation.

## Help text principle

Every command should have useful examples.

Avoid help text that only repeats the command name.
