# Test Strategy

## Goals

Kitout should be safe to run repeatedly. Tests should prove that resources check status correctly, plan safely, and apply only intended changes.

## Test levels

### Unit tests

Test each resource independently.

Required cases:

- satisfied state
- missing state
- changed state
- failed state
- dry-run behavior
- command construction
- error messages

### Config tests

Test:

- valid config
- missing version
- unknown top-level key
- unknown resource field
- duplicate resources
- invalid paths
- invalid macOS default type

### Engine tests

Test:

- status aggregation
- plan building
- dry-run does not call apply
- apply order
- partial failure handling
- exit code mapping

### CLI tests

Test:

- command flags
- output summaries
- JSON output
- config path selection
- help text examples

## Command runner abstraction

External commands must go through an interface.

```go
type Runner interface {
    Run(ctx context.Context, name string, args ...string) (CommandResult, error)
}
```

Tests should use a fake runner.

Do not call real `brew`, `asdf`, `git`, `defaults`, or `ln` in unit tests.

## Fixture structure

```txt
testdata/
  configs/
    valid.basic.yaml
    invalid.unknown-field.yaml
    invalid.duplicate.yaml
  outputs/
    status.basic.txt
    apply.dry-run.txt
```

## Safety regression tests

Add explicit tests that prove:

- status does not apply changes
- dry-run does not apply changes
- symlink replacement defaults to false
- copy replacement defaults to false and does not follow symlinked sources
- asdf dry-run plans do not add plugins, install versions, or write `.tool-versions`
- shell command resources require explicit config
- unknown config fields fail validation

## Release gate

Before release, run the macOS-local release gate:

```sh
make release-check
```

The target checks formatting, runs `go test ./...` and `go vet ./...`, then
runs `make smoke-distribution`. The smoke target builds `bin/kitout`, creates a
temporary HOME, writes starter configs with `kitout init --config`, runs
`kitout doctor`, expects `kitout status` to report the missing starter
directories such as `~/code`, verifies `kitout apply --dry-run` exits without
changing the temporary filesystem, applies a temp-only nested directory copy,
and checks a safe login-shell status/dry-run plan when the host can report the
current user's `UserShell`, without calling `chsh` or editing `/etc/shells`.

Then test on a disposable macOS user account if possible.
