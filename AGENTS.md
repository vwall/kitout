# AGENTS.md

This file guides Codex and other coding agents working in the Kitout repository.

## Project summary

Kitout is a Go CLI for declaratively setting up a Mac. It manages system resources such as Homebrew packages, casks, directories, symlinks, Git repositories, macOS defaults, and approved shell commands.

The project is inspired by Bork, but it should not be a Bash DSL parser or a strict compatibility layer. Build a clean Go-native tool with a structured config format.

## Current phase

Phase 0: documentation, architecture, and core design.

Do not build a large framework yet. Prefer small, testable packages and a boring command surface.

## Product goals

Kitout should let a user run:

```sh
kitout status
kitout apply --dry-run
kitout apply
kitout doctor
```

The output should clearly show:

- what is already satisfied
- what is missing
- what will change
- what failed
- how to fix common problems

## Non-goals for the MVP

Do not implement these in the first pass unless explicitly requested:

- Linux support
- Windows support
- secret management
- templating
- plugin systems
- remote config fetching
- graphical user interface
- AI features
- Borkfile compatibility
- arbitrary shell DSL parsing

## Preferred implementation direction

Use Go.

Recommended structure:

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

internal/engine/
  resource.go
  planner.go
  executor.go
  result.go

internal/resources/
  brew.go
  cask.go
  directory.go
  symlink.go
  repo.go
  macos_default.go
  shell.go

internal/platform/
  macos.go
  command.go

testdata/
  kitout.basic.yaml
```

## Core abstractions

A resource checks and satisfies one unit of desired state.

```go
type Resource interface {
    ID() string
    Type() string
    Status(ctx context.Context) (StatusResult, error)
    Apply(ctx context.Context) (ApplyResult, error)
}
```

Status should never make changes.

Apply should be idempotent.

Dry-run should never make changes.

## Config rules

The first config format is YAML.

The default config path is:

```txt
~/.config/kitout/kitout.yaml
```

A repo-local config may also be supported:

```txt
./kitout.yaml
```

If both exist, use explicit CLI flags to avoid surprising behavior.

## Command expectations

### `kitout status`

Checks all resources and reports satisfied, missing, changed, failed, and skipped resources.

### `kitout apply`

Plans and applies missing or incorrect resources.

### `kitout apply --dry-run`

Shows what would change without changing anything.

### `kitout doctor`

Checks prerequisites, including macOS version, Homebrew, Git, shell, config validity, and path permissions.

### `kitout init`

Creates a starter config file and optional example folder structure.

## Testing expectations

Each resource must have unit tests for:

- satisfied state
- missing state
- apply success
- apply failure
- dry-run behavior
- command construction

External commands must be wrapped behind an interface so tests do not call real Homebrew, Git, or macOS defaults.

## Documentation expectations

When adding a feature, update:

- relevant docs in `docs/`
- example config in `examples/`
- CLI help text
- tests

## Style preferences

- Keep code simple.
- Prefer explicit types over clever abstractions.
- Keep resources independent.
- Avoid global state.
- Avoid hidden mutation during planning.
- Return structured results, then render them in the CLI layer.
- Use clear error messages.

## Safety rules

Never delete or overwrite user files unless the config explicitly allows it.

Never store secrets in the config.

Never run shell commands by default unless they are explicitly listed in the config.

Never auto-install Homebrew without asking or requiring an explicit flag.

Never modify shell profiles without showing the target path and intended content.

## Commit guidance

Prefer small commits. Good commit examples:

```txt
Add resource interface and status result types
Add config loader and validation errors
Add brew package resource status check
Add CLI status command renderer
Document initial YAML config schema
```

Avoid mixed commits that change docs, CLI behavior, config schema, and resource logic all at once.

# Project Context

Linear team: BUILD
Linear initiative: Kitout

Use Linear for private planning.
When creating Linear issues from this repo, associate them with the Kitout initiative or a Kitout project when available.
