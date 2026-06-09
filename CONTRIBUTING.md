# Contributing

## Development setup

```sh
go mod download
make test
go run ./cmd/kitout version
make build
```

## Before opening a pull request

Run:

```sh
go test ./...
go vet ./...
gofmt -w .
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

## Design changes

Use an RFC for changes to:

- config schema
- command behavior
- resource interface
- safety defaults
- output format

## Commit style

Prefer clear commit messages:

```txt
Add directory resource status checks
Validate duplicate symlink targets
Document apply dry-run behavior
```
