# Runtime Resource Model

## Core idea

A resource represents one desired piece of machine state.

Examples:

- `brew:git`
- `asdf_plugin:ruby`
- `asdf_tool_versions:~/.tool-versions`
- `cask:ghostty`
- `directory:~/code`
- `symlink:~/.zshrc`
- `repo:~/code/example-project`
- `macos_default:NSGlobalDomain/AppleShowAllExtensions`

Every resource can be checked. Some resources can be applied.

## Interface

```go
type Resource interface {
    ID() string
    Type() string
    Status(ctx context.Context) (StatusResult, error)
    Apply(ctx context.Context) (ApplyResult, error)
}
```

## StatusResult

```go
type ResourceState string

const (
    StateSatisfied ResourceState = "satisfied"
    StateMissing   ResourceState = "missing"
    StateChanged   ResourceState = "changed"
    StateFailed    ResourceState = "failed"
    StateSkipped   ResourceState = "skipped"
    StateUnknown   ResourceState = "unknown"
)

type StatusResult struct {
    ResourceID string
    Type       string
    State      ResourceState
    Message    string
    Details    map[string]string
}
```

## ApplyResult

```go
type ApplyResult struct {
    ResourceID string
    Type       string
    Action     string
    Changed    bool
    Message    string
    Details    map[string]string
}
```

## Required properties

### Idempotent

Running apply twice should not apply the same change twice.

### Observable

Status should be able to explain what it found.

### Safe

Status and dry-run must never modify the system.

### Local

Resources should operate on the local machine only in the MVP.

## Resource IDs

IDs should be stable and human-readable.

Examples:

```txt
brew:git
asdf_plugin:ruby
asdf_tool_versions:/Users/example/.tool-versions
cask:visual-studio-code
directory:/Users/example/code
symlink:/Users/example/.zshrc
repo:/Users/example/code/example-project
macos_default:NSGlobalDomain/AppleShowAllExtensions
```

## Planner

The planner turns status results into actions.

```txt
satisfied -> no-op
missing -> apply
changed -> apply if safe, otherwise warn
failed -> fail
skipped -> skip
unknown -> fail unless explicitly allowed
```

## Executor

The executor runs actions in order.

MVP execution should be sequential. Parallel execution can come later.

Sequential execution makes output easier to follow and avoids command conflicts, especially with Homebrew.

## Dependency handling

Do not build a full dependency graph in the MVP.

Use fixed execution order:

1. doctor prerequisites
2. Homebrew packages
3. asdf plugins and versions
4. asdf `.tool-versions` entries
5. casks
6. directories
7. repositories
8. symlinks
9. macOS defaults
10. shell commands

This is enough for a first version.

## Dry-run behavior

Dry-run should:

- load config
- validate config
- check statuses
- build a plan
- render intended changes
- make no changes

Dry-run should not:

- install packages
- clone repositories
- create directories
- create symlinks
- run shell commands
- write files

## Exit codes

Recommended exit codes:

```txt
0 all satisfied or apply completed successfully
1 changes needed when running status
2 validation error
3 runtime error
4 partial apply failure
```
