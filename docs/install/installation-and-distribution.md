# Installation and Distribution

## Development install

During development:

```sh
go install ./cmd/kitout
```

After installing it onto your `PATH`, run:

```sh
kitout status
```

When working from source without installing, use `go run` as the
developer-local equivalent:

```sh
go run ./cmd/kitout status
```

For a real setup repo, keep the config path explicit while dogfooding:

```sh
kitout doctor --config ~/code/setup/kitout.yaml
kitout status --config ~/code/setup/kitout.yaml
kitout apply --config ~/code/setup/kitout.yaml --dry-run
```

See `docs/setup/first-real-run.md` for the full first-run checklist.

To produce a local build artifact with embedded metadata:

```sh
make build
bin/kitout version
```

## Release goals

Kitout should eventually support:

```sh
brew install kitout
```

or:

```sh
brew tap vwall/kitout
brew install kitout
```

## Distribution readiness gate

Before starting distribution work, fix or verify these items:

- release builds produce darwin arm64 and amd64 binaries with embedded version,
  commit, and build date metadata
- release artifacts include checksums generated from the exact binaries that will
  be published
- `go test ./...` passes from a clean checkout
- `make smoke-distribution` passes on macOS, covering a temporary config,
  `init`, `doctor`, `status`, and `apply --dry-run`
- `README.md`, `docs/config/config-spec.md`, `docs/cli/cli-spec.md`, and
  `examples/kitout.yaml` match the current command behavior and config schema
- `kitout init` writes a starter config that validates without manual edits and
  `kitout status --config <generated-file>` can parse through it immediately
- the generated starter keeps external-command resources commented until the
  user replaces placeholder package, repo, and dotfile values
- risky apply actions still require confirmation unless `--yes` is passed
- fresh-machine prerequisite friction is documented in
  `docs/setup/first-real-run.md`

These are not blockers for the first distribution:

- automatic Homebrew installation
- automatic repo-local config selection
- Linux or Windows builds
- config includes, templates, secrets, or plugins
- a larger macOS defaults library
- shell profile mutation helpers

## Homebrew tap

A Homebrew tap is the natural distribution method for a developer-focused macOS CLI.

Recommended tap repo:

```txt
github.com/vwall/homebrew-kitout
```

Formula name:

```txt
kitout
```

## GitHub releases

Each release should include:

- macOS arm64 binary
- macOS amd64 binary
- checksums
- changelog

Future releases may include Linux builds.

## Versioning

Use semantic versioning once the config schema has users.

Before that, use `0.x.y` versions.

Example:

```txt
0.1.0 initial MVP
0.2.0 macOS defaults resource
0.3.0 JSON output
1.0.0 stable config schema
```

## Build metadata

The binary should expose:

```txt
version
commit
build date
```

Example:

```sh
kitout version
```

Local and release builds should set metadata with Go linker flags. The stable
linker variables are:

```txt
github.com/vwall/kitout/internal/buildinfo.Version
github.com/vwall/kitout/internal/buildinfo.Commit
github.com/vwall/kitout/internal/buildinfo.BuildDate
```

The local build target injects these values automatically:

```sh
make build VERSION=0.1.0
```

Release tooling should use the same variables:

```sh
go build -trimpath \
  -ldflags "-s -w -X github.com/vwall/kitout/internal/buildinfo.Version=0.1.0 -X github.com/vwall/kitout/internal/buildinfo.Commit=$(git rev-parse --short HEAD) -X github.com/vwall/kitout/internal/buildinfo.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o dist/kitout ./cmd/kitout
```

## First install docs

A future public README should include:

```sh
brew tap vwall/kitout
brew install kitout
kitout init
kitout status
```

## Private usage

The project can start private.

A private setup repo can contain:

```txt
kitout.yaml
dotfiles/
scripts/
README.md
```

Kitout itself can be public or private. The user's setup repo should remain private if it contains personal paths, repo lists, or machine preferences.

When the setup repo contains `kitout.yaml`, the current CLI still uses
`~/.config/kitout/kitout.yaml` by default. Pass `--config ./kitout.yaml` from
inside the setup repo, or copy the file to the default path when the setup is
ready to become the normal machine config.
