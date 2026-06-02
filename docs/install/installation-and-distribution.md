# Installation and Distribution

## Development install

During development:

```sh
go install ./cmd/kitout
```

or:

```sh
go run ./cmd/kitout status
```

## Release goals

Kitout should eventually support:

```sh
brew install kitout
```

or:

```sh
brew tap vrwaller/tap
brew install kitout
```

## Homebrew tap

A Homebrew tap is the natural distribution method for a developer-focused macOS CLI.

Recommended tap repo:

```txt
github.com/vrwaller/homebrew-tap
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

## First install docs

A future public README should include:

```sh
brew tap vrwaller/tap
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
