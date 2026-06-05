# Kitout

Kitout is a Go-based command line tool for setting up a fresh Mac with the packages, apps, repositories, dotfiles, folders, and system preferences that make it feel like home.

Kitout is inspired by Bork's assertion-based approach, but it is not a strict port. The goal is to keep the useful idea of declaring desired machine state while using a typed Go codebase, a structured config file, better test coverage, and a clearer command surface.

## Install

The first public release target is `v0.1.0` through GitHub releases and a Homebrew tap:

```sh
brew tap vwall/kitout
brew install kitout
```

Until the tap is published, build from source with:

```sh
go install ./cmd/kitout
```

## Product statement

> Kitout equips a fresh Mac with your apps, packages, repos, dotfiles, and defaults.

## Primary command flow

After installing Kitout, run:

```sh
kitout init
kitout status
kitout apply --dry-run
kitout apply
kitout doctor
```

When using a cloned setup repo before installing a config to the default path,
pass the config explicitly:

```sh
kitout doctor --config ~/code/setup/kitout.yaml
kitout status --config ~/code/setup/kitout.yaml
kitout apply --config ~/code/setup/kitout.yaml --dry-run
```

See `docs/setup/first-real-run.md` for the practical first-run loop and common
fresh-machine friction.

## Example config

```yaml
version: 1

brew:
  packages:
    - git
    - asdf
    - pnpm
    - gh

asdf:
  plugins:
    - name: ruby
      url: https://github.com/asdf-vm/asdf-ruby.git
      versions:
        - 3.3.6
  tool_versions:
    - path: ~/.tool-versions
      tools:
        ruby: 3.3.6

casks:
  - ghostty
  - visual-studio-code
  - rectangle

directories:
  - ~/code
  - ~/.config

repos:
  - path: ~/code/example-project
    url: git@github.com:example/example-project.git

symlinks:
  - source: ~/dotfiles/home/zshrc
    target: ~/.zshrc

symlink_groups:
  - source_root: ~/dotfiles/home
    target_root: ~
    paths:
      - .gitconfig
      - .config/ghostty
```

## Initial scope

The first version should focus on macOS, Apple Silicon, Homebrew, asdf-managed developer runtimes, Git repositories, directories, symlinks, shell commands, and a safe dry-run mode.

Do not start with Linux support, secrets, templates, plugins, or a package manager abstraction. Those can come later.

## Documentation map

- `docs/charter/project-charter.md`
- `docs/product/product-brief.md`
- `docs/product/naming-positioning.md`
- `docs/architecture/architecture-overview.md`
- `docs/runtime/resource-model.md`
- `docs/cli/cli-spec.md`
- `docs/config/config-spec.md`
- `docs/setup/first-real-run.md`
- `docs/resources/*.md`
- `docs/install/installation-and-distribution.md`
- `docs/testing/test-strategy.md`
- `docs/governance/rfcs.md`
- `rfcs/*.md`
- `AGENTS.md`

## Development

Run the installed CLI with:

```sh
kitout version
kitout init
kitout status
kitout apply --dry-run
```

When working from source, use `go run` as the developer-local equivalent:

```sh
go run ./cmd/kitout version
go run ./cmd/kitout init
go run ./cmd/kitout status
go run ./cmd/kitout apply --dry-run
```

Run tests with:

```sh
go test ./...
```

Build a local binary with embedded version metadata:

```sh
make build
bin/kitout version
```

## Working principles

1. Desired state over imperative scripting.
2. Safe previews before changes.
3. Idempotent resource application.
4. Human-readable output.
5. Small core before extensibility.
6. Mac-first, not Mac-only forever.
