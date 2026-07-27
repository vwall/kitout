# Contributing

## Development setup

```sh
go mod download
make fmt-check
make test
go run ./cmd/kitout version
make build
```

## Before opening a pull request

Run:

```sh
make fmt-check
make test
make vet
```

Before cutting a release on macOS, run:

```sh
make release-check
```

## Pull request expectations

A good pull request should include:

- focused implementation
- tests
- docs update when behavior changes
- example config update when schema changes

## Broad changes

Open a GitHub issue before implementing changes to:

- config schema
- command behavior
- resource interface
- safety defaults
- output format

Describe the user problem, proposed CLI or config shape, compatibility impact,
and how status, dry-run, and apply remain safe and idempotent.

## Commit style

Prefer clear commit messages:

```txt
Add directory resource status checks
Validate duplicate symlink targets
Document apply dry-run behavior
```
