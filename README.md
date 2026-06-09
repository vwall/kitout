# Kitout

Kitout is a Go-based command line tool for setting up a fresh Mac with the packages, apps, repositories, file copies, dotfiles, folders, login shell, and system preferences that make it feel like home.

Kitout is inspired by Bork's assertion-based approach, but it is not a strict port. The goal is to keep the useful idea of declaring desired machine state while using a typed Go codebase, a structured config file, better test coverage, and a clearer command surface.

## Install

The public release install path is GitHub releases plus the Homebrew tap:

```sh
brew tap vwall/kitout
brew install kitout
```

To build from source instead:

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

When using a cloned setup repo with `kitout.yaml` at its root, run Kitout from
that directory. If you also have `~/.config/kitout/kitout.yaml`, pass
`--config ./kitout.yaml` so Kitout knows you intend to trust the repo-local
file:

```sh
cd ~/code/setup
kitout doctor --config ./kitout.yaml
kitout status --config ./kitout.yaml
kitout apply --config ./kitout.yaml --dry-run
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

login_shell:
  path: homebrew:fish
  add_to_etc_shells: true

directories:
  - ~/code
  - ~/.config
  - ~/.codex/skills

repos:
  - path: ~/code/example-project
    url: git@github.com:example/example-project.git

copies:
  - source: ./codex/skills/nuxt-practices
    target: ~/.codex/skills/nuxt-practices
    replace: false

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

## Current scope

Kitout 1.1.x is macOS-focused and covers Apple Silicon, Homebrew, asdf-managed developer runtimes, Git repositories, directories, file copies, symlinks, login shell management, shell commands, and a safe dry-run mode.

Do not start with Linux support, secrets, templates, plugins, or a package manager abstraction. Those can come later.

## Documentation map

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

## Issues

Create a GitHub issue for bugs, feature requests, support questions, and
security reports: `https://github.com/vwall/kitout/issues/new`.

Use GitHub private vulnerability reporting when it is enabled and the report
contains sensitive security details.

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
make test
```

Build a local binary with embedded version metadata:

```sh
make build
bin/kitout version
```

Before cutting a release on macOS, run the full release gate:

```sh
make release-check
```

## Working principles

1. Desired state over imperative scripting.
2. Safe previews before changes.
3. Idempotent resource application.
4. Human-readable output.
5. Small core before extensibility.
6. Mac-first, not Mac-only forever.
