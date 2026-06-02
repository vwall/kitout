# Kitout

Kitout is a Go-based command line tool for setting up a fresh Mac with the packages, apps, repositories, dotfiles, folders, and system preferences that make it feel like home.

Kitout is inspired by Bork's assertion-based approach, but it is not a strict port. The goal is to keep the useful idea of declaring desired machine state while using a typed Go codebase, a structured config file, better test coverage, and a clearer command surface.

## Product statement

> Kitout equips a fresh Mac with your apps, packages, repos, dotfiles, and defaults.

## Primary command flow

```sh
kitout init
kitout status
kitout apply
kitout doctor
```

## Example config

```yaml
version: 1

brew:
  packages:
    - git
    - ruby
    - node
    - pnpm
    - gh

casks:
  - ghostty
  - visual-studio-code
  - 1password

directories:
  - ~/code
  - ~/.config

repos:
  - path: ~/code/aubs
    url: git@github.com:vrwaller/aubs.git

symlinks:
  - source: ~/dotfiles/home/zshrc
    target: ~/.zshrc
```

## Initial scope

The first version should focus on macOS, Apple Silicon, Homebrew, Git repositories, directories, symlinks, shell commands, and a safe dry-run mode.

Do not start with Linux support, secrets, templates, plugins, or a package manager abstraction. Those can come later.

## Documentation map

- `docs/charter/project-charter.md`
- `docs/product/product-brief.md`
- `docs/product/naming-positioning.md`
- `docs/architecture/architecture-overview.md`
- `docs/runtime/resource-model.md`
- `docs/cli/cli-spec.md`
- `docs/config/config-spec.md`
- `docs/resources/*.md`
- `docs/install/installation-and-distribution.md`
- `docs/testing/test-strategy.md`
- `docs/governance/rfcs.md`
- `rfcs/*.md`
- `AGENTS.md`

## Development

Run the CLI locally with:

```sh
go run ./cmd/kitout version
go run ./cmd/kitout init --config ./kitout.yaml
```

Run tests with:

```sh
go test ./...
```

## Working principles

1. Desired state over imperative scripting.
2. Safe previews before changes.
3. Idempotent resource application.
4. Human-readable output.
5. Small core before extensibility.
6. Mac-first, not Mac-only forever.
