# CLI Specification

## Command name

```sh
kitout
```

## Global flags

```txt
--config PATH       Path to config file (default: ~/.config/kitout/kitout.yaml)
--verbose           Show detailed command output
--quiet             Reduce output
--no-color          Disable colored output
--json              Print machine-readable JSON output
--yes               Skip interactive confirmations when allowed
```

These flags are currently parsed by the root command and by implemented
subcommands. `init`, `doctor`, `status`, and `apply` use
`~/.config/kitout/kitout.yaml` by default; pass `--config` only to override that
path. `--json` and `--quiet` currently affect `status` and `apply`.

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
- write a commented starter config

### `kitout status`

Checks configured resources.

```sh
kitout status
kitout status --json
kitout status --config ./kitout.yaml
```

Current behavior:

- load and validate the selected config file
- build resources from config in stable execution order
- check resource status through the engine planner
- render resource details and summary counts
- return `0` when all resources are satisfied or skipped
- return `1` when changes are needed
- return `2` for config validation, parse, or unknown-field errors
- return `3` for config read failures or failed/unknown resources
- when `--json` is passed, print machine-readable plan output

Example human output:

```txt
Config: /Users/example/.config/kitout/kitout.yaml
ok   brew:git           formula is installed
need cask:ghostty       cask is missing
ok   directory:/Users/example/code directory exists
need symlink:/Users/example/.zshrc symlink points elsewhere

2 changes needed
```

Exit behavior:

```txt
0 all resources satisfied
1 changes needed
2 validation error
3 runtime error
```

### `kitout apply`

Applies needed changes.

```sh
kitout apply
kitout apply --dry-run
kitout apply --yes
```

Behavior:

- load and validate config
- run status checks
- build plan
- apply missing or changed resources in stable order
- render summary
- return `0` when apply completed successfully
- return `2` for validation, parse, unknown-field, or flag errors
- return `3` for config read failures or pre-apply plan failures
- return `4` for partial apply failures

### `kitout apply --dry-run`

Shows intended changes without making changes.

Example output:

```txt
Config: /Users/example/.config/kitout/kitout.yaml
Plan:
  apply cask:ghostty       cask is missing
  apply symlink:/Users/example/.zshrc symlink points elsewhere

No changes made because --dry-run was used.
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
- Git installation
- config file validity

Current output:

```txt
Config: /Users/me/.config/kitout/kitout.yaml
Doctor:
ok   macOS                      running on macOS
ok   CPU architecture           running on Apple Silicon
ok   Xcode Command Line Tools   /Library/Developer/CommandLineTools
ok   Homebrew                   Homebrew 4.0.0
ok   Git                        git version 2.45.0
ok   Config                     config is valid

6 total, 6 ok, 0 warnings, 0 failed
```

Planned follow-up checks:

- write permissions for target paths
- Homebrew path
- shell environment

### `kitout list`

Optional MVP command.

Shows resources parsed from config without checking the system.

```sh
kitout list
```

### `kitout version`

Prints version metadata.

```sh
kitout version
```

Current output:

```txt
kitout 0.1.0
commit unknown
built unknown
```

Release builds may set `commit` and `built` metadata at build time.

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

Interactive confirmation is planned for risky changes unless `--yes` is passed.

Risky changes include:

- replacing symlink targets
- modifying macOS defaults
- running shell commands

Once confirmation prompts are implemented, safe changes may apply without
confirmation:

- installing a missing package
- creating a missing directory
- cloning a missing repo

## Help text principle

Every command should have useful examples.

Avoid help text that only repeats the command name.
