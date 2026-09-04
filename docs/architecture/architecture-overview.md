# Architecture Overview

## Summary

Kitout is a local-first Go CLI. It reads a YAML configuration file, converts it into resource definitions, checks current system state, creates a plan, and optionally applies that plan.

## High-level flow

```txt
CLI command
  -> config loader
  -> config validator
  -> resource builder
  -> planner
  -> executor
  -> renderer
```

## Layers

### CLI layer

Responsible for:

- parsing commands and flags
- loading config path options
- selecting status, dry-run, apply, doctor, or init behavior
- rendering output
- returning process exit codes

The CLI layer should not know how Homebrew or symlinks work.

The CLI composition root owns process-scoped dependencies: cancellation
context, standard streams, and command-runner factories. Commands receive
those dependencies through the application value instead of package globals,
so command work is cancelable and tests can isolate runner implementations.

### Config layer

Responsible for:

- reading YAML
- expanding paths
- validating required fields
- detecting duplicate resources
- converting config into typed resource specs

The config layer should not run system commands.

### Engine layer

Responsible for:

- accepting resources
- checking status
- building plans
- executing plans
- collecting results
- enforcing dry-run safety

The engine layer should not render terminal UI directly.

### Resource layer

Responsible for resource-specific logic.

Examples:

- brew package
- asdf plugin
- asdf tool versions
- cask
- directory
- copy
- symlink
- Git repository
- macOS default
- security setting
- system prerequisite
- SSH key
- shell command

Each resource must implement the same lifecycle.

### Platform layer

Responsible for OS-specific helpers:

- command execution
- path expansion
- macOS version detection
- Homebrew path detection
- shell detection

Command execution bounds waits on inherited subprocess pipes. On Darwin and
Linux, every command runs in a private process group so cancellation and final
cleanup include descendants created before the command leader exits. For
interactive commands, Kitout temporarily makes that private group the
terminal's foreground group so Git, `chsh`, nested `sudo`, and configured shell
commands can use `/dev/tty`. A small process-group supervisor forwards terminal
interrupts to Kitout, ensuring context cancellation still begins when a command
ignores `SIGINT`. It also coordinates terminal stop and continue signals so
suspending Kitout and returning it to the foreground resumes the command group
with terminal ownership intact. Kitout restores its own foreground group when
the foreground command finishes, while background completion leaves the shell's
terminal ownership unchanged. Canceling first gives the owned group a bounded
chance to relay termination across a privilege boundary, then forcibly
terminates that group.

## Package layout

```txt
cmd/kitout/
  main.go

internal/cli/
  root.go
  status.go
  apply.go
  doctor.go
  init.go

internal/config/
  config.go
  loader.go
  validate.go
  paths.go

internal/engine/
  resource.go
  planner.go
  executor.go
  result.go

internal/resources/
  brew.go
  cask.go
  directory.go
  copy.go
  symlink.go
  repo.go
  macos_default.go
  security.go
  system.go
  ssh_key.go
  shell.go

internal/platform/
  command.go
  macos.go
  homebrew.go
  paths.go

examples/
  kitout.yaml

testdata/
  kitout.basic.yaml
```

## Data flow

### Status

```txt
Load config
Validate config
Build resources
For each resource, call Status
Render results
Exit 0 if all satisfied
Exit 1 if changes are needed
Exit 2 if errors occurred
```

### Apply

```txt
Load config
Validate config
Build resources
Check status
Build plan
If dry-run, render plan and exit
Apply each planned change
Render results
Exit according to result state
```

## State model

Resource states:

- satisfied
- missing
- changed
- failed
- skipped
- unknown

Plan actions:

- no-op
- create
- install
- update
- replace
- run
- warn

## Planning inspections

Planning resources share Homebrew inventories and asdf version/plugin inventory
checks within one resource build. Each new plan builds a fresh resource list;
there is no persistent cache. Per-plugin asdf version checks remain live.
Apply and upgrade use uncached resources, and Homebrew update rechecks target
the selected package rather than scanning the entire installed inventory.

## Error philosophy

Errors should include:

- resource ID
- resource type
- action attempted
- underlying command or path when safe to show
- suggestion when possible

Example:

```txt
Failed: symlink ~/.zshrc
Reason: target exists and points to /tmp/old-zshrc
Suggestion: rerun with --replace-conflicts if you intentionally want Kitout to replace it
```

## Safety design

Kitout should default to no destructive actions.

For risky resources, require explicit config fields or CLI flags.

Examples:

```yaml
symlinks:
  - source: ~/dotfiles/home/zshrc
    target: ~/.zshrc
    replace: false
```

## Extensibility

Do not build a plugin system in the MVP.

Instead, keep resources small and consistent so adding a new built-in resource is simple.
