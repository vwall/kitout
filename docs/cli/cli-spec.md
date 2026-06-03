# CLI Specification

## Command name

```sh
kitout
```

## Global flags

```txt
--config PATH       Path to config file
--verbose           Show detailed command output
--quiet             Reduce output
--no-color          Disable colored output
--json              Print machine-readable JSON output
--yes               Skip interactive confirmations when allowed
```

These flags are currently parsed by the root command and by implemented
subcommands. `--config` is used by `init` and `status`; `--json` and `--quiet`
currently affect `status`.

## Commands

### `kitout init`

Creates a starter config file.

```sh
kitout init
kitout init --config ~/.config/kitout/kitout.yaml
```

Behavior:

- create parent config directory if missing
- refuse to overwrite existing config unless `--force` is passed
- write a commented starter config

### `kitout status`

Checks configured resources.

```sh
kitout status
kitout status --config ./kitout.yaml
kitout status --json
```

Current behavior:

- load and validate the selected config file
- return `2` for config validation, parse, or unknown-field errors
- return `3` for config read failures
- do not check resource status until the CLI is wired to the engine resources
- when `--json` is passed, print a small machine-readable response for config load, validation, and the current unimplemented status state

Current human output for a valid config:

```txt
Config valid: /Users/example/.config/kitout/kitout.yaml
Status checks are not implemented yet.
```

Target output after resource checks are wired:

```txt
✓ brew package git installed
✓ cask visual-studio-code installed
✗ cask ghostty missing
✓ directory ~/code exists
! symlink ~/.zshrc points to a different source

2 changes needed
```

Target exit behavior:

```txt
0 all resources satisfied
1 changes needed
2 validation error
3 runtime error
```

Current exit behavior:

```txt
0 config is valid
2 validation, parse, unknown-field, or flag error
3 config read or runtime render error
```

### `kitout apply`

Planned command.

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
- ask for confirmation if needed
- apply changes in stable order
- render summary

### `kitout apply --dry-run`

Planned command.

Shows intended changes without making changes.

Expected output:

```txt
Plan
  install cask ghostty
  replace symlink ~/.zshrc with ~/dotfiles/home/zshrc

No changes made because --dry-run was used.
```

### `kitout doctor`

Planned command.

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

Current `status --json` output only reports config validity and whether resource
status checks are implemented. It does not include resource status summaries
until the CLI is wired to the engine resources.

Example:

```json
{
  "command": "status",
  "ok": true,
  "config": {
    "path": "/Users/example/.config/kitout/kitout.yaml",
    "valid": true
  },
  "status": {
    "implemented": false,
    "message": "Status checks are not implemented yet."
  }
}
```

## Confirmation rules

Interactive confirmation should be required for risky changes unless `--yes` is passed.

Risky changes include:

- replacing symlink targets
- modifying macOS defaults
- running shell commands

Safe changes may apply without confirmation:

- installing a missing package
- creating a missing directory
- cloning a missing repo

## Help text principle

Every command should have useful examples.

Avoid help text that only repeats the command name.
