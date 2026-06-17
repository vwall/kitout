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

For a real setup repo that contains `kitout.yaml`, run Kitout from the repo root
while dogfooding. If `~/.config/kitout/kitout.yaml` also exists, pass
`--config ./kitout.yaml`:

```sh
cd ~/code/setup
kitout doctor --config ./kitout.yaml
kitout status --config ./kitout.yaml
kitout apply --config ./kitout.yaml --dry-run
```

See `docs/setup/first-real-run.md` for the full first-run checklist.

To produce a local build artifact with embedded metadata:

```sh
make build
bin/kitout version
```

## Release baseline

Kitout is distributed through GitHub releases and an external Homebrew tap. The
project is now past the `v1.1.2` patch release and remains in the stable
`version: 1` config schema line.

The intended Homebrew install path is:

```sh
brew tap vwall/kitout
brew install kitout
```

## Distribution readiness gate

Before tagging a release on macOS, run:

```sh
make release-check VERSION=X.Y.Z
```

The release gate runs `go test ./...`, then `go vet ./...`, then
`make smoke-distribution`. The smoke target builds `bin/kitout` and covers
temporary starter configs, `init`, `doctor`, `status`, `apply --dry-run`, a
temp-only directory copy apply, and a safe login-shell status/dry-run plan.

Before tagging, also verify these release-specific items:

- release builds produce darwin arm64 and amd64 binaries with embedded version,
  commit, and build date metadata
- release artifacts include checksums generated from the exact binaries that will
  be published
- `README.md`, `docs/config/config-spec.md`, `docs/cli/cli-spec.md`, and
  `examples/kitout.yaml` match the current command behavior and config schema
- `kitout init` writes a starter config that validates without manual edits and
  `kitout status --config <generated-file>` can parse through it immediately
- the generated starter keeps non-directory resources commented until the user
  replaces placeholder package, repo, copy, dotfile, and login-shell values
- risky apply actions still require confirmation unless `--yes` is passed
- copy replacements and login-shell changes are still treated as risky apply
  actions
- fresh-machine prerequisite friction is documented in
  `docs/setup/first-real-run.md`

If Go tries to write module or build-cache files in non-writable user cache
directories during a sandboxed release check, rerun the gate with writable Go
caches:

```sh
tmp_gomodcache="$(mktemp -d)"
tmp_gocache="$(mktemp -d)"
GOMODCACHE="$tmp_gomodcache" GOCACHE="$tmp_gocache" make release-check VERSION=X.Y.Z
```

This keeps the release check quiet without changing the published artifacts.

These remain outside the current release scope:

- automatic Homebrew installation
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

- `kitout_<version>_darwin_arm64.tar.gz`
- `kitout_<version>_darwin_amd64.tar.gz`
- `kitout_<version>_checksums.txt`
- changelog

For a `vX.Y.Z` release, publish:

- `kitout_X.Y.Z_darwin_arm64.tar.gz`
- `kitout_X.Y.Z_darwin_amd64.tar.gz`
- `kitout_X.Y.Z_checksums.txt`

Future releases may include Linux builds.

The GitHub Actions release workflow runs when a `v*.*.*` tag is pushed. It
builds both macOS binaries from `./cmd/kitout`, embeds release metadata with the
same linker variables as local builds, runs `make release-check`, packages the
binaries, writes checksums, and creates the GitHub release. If
`docs/release/<version>.md` exists, where `<version>` is the tag without the
leading `v`, the workflow uses that file as the release body. If not, it uses
`docs/release/<tag>.md` when present, such as `docs/release/vX.Y.Z.md`. When
neither file exists, the workflow publishes a small generated fallback body that
lists the release assets.

The archive layout is stable:

```txt
kitout_<version>_darwin_<arch>/
  kitout
```

Checksums are generated after packaging from the exact `*.tar.gz` files that
are uploaded as release assets. Do not rebuild binaries or regenerate archives
between checksum generation and upload.

To publish a release:

```sh
git tag vX.Y.Z
git push origin vX.Y.Z
```

After the GitHub release is created, update the tap formula from
`packaging/homebrew/kitout.rb.template` with the generated checksums from
`kitout_X.Y.Z_checksums.txt`, then publish it to
`github.com/vwall/homebrew-kitout`. For each new release, update the template
version and asset URLs to the new `vX.Y.Z` artifacts before filling the checksum
placeholders from the published tarballs.

## Versioning

Use semantic versioning. The `1.0.0` release establishes config schema
`version: 1` as stable.

Example:

```txt
0.1.0 initial MVP
0.1.1 safer setup-repo config discovery
1.0.0 stable config schema and first Homebrew release
1.1.0 additive config/resource features
1.1.1 release polish and smoke coverage
1.1.2 fresh-machine apply continuation
2.0.0 breaking config schema changes
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
make build VERSION=X.Y.Z
```

Release tooling should use the same variables:

```sh
go build -trimpath \
  -ldflags "-s -w -X github.com/vwall/kitout/internal/buildinfo.Version=X.Y.Z -X github.com/vwall/kitout/internal/buildinfo.Commit=$(git rev-parse --short HEAD) -X github.com/vwall/kitout/internal/buildinfo.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o dist/kitout ./cmd/kitout
```

Tagged GitHub releases strip the leading `v` from the tag before embedding the
version. A `vX.Y.Z` tag therefore produces binaries whose `kitout version`
output starts with `kitout X.Y.Z`.

## First install docs

Public install docs should include:

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

When the setup repo contains `kitout.yaml`, run Kitout from inside the setup
repo and pass `--config ./kitout.yaml` whenever a home config also exists. Copy
the file to `~/.config/kitout/kitout.yaml` only when the setup is ready to
become the normal machine config outside the repo.
