# Brew Package Resource

## Purpose

The brew package resource ensures a Homebrew formula is installed.

## Config

```yaml
brew:
  packages:
    - git
    - ruby
    - node
```

Future expanded form:

```yaml
brew:
  packages:
    - name: git
    - name: ruby
      version: latest
```

## Status check

Use:

```sh
brew list --formula <name>
brew outdated --formula --quiet <name>
```

Satisfied when the formula is installed.

Missing when Homebrew is available but the formula is not installed.

Changed when the formula is installed and Homebrew reports that it is outdated.
Human status output marks this as `changed`.

Some Homebrew versions return exit code 1 from `brew outdated` when the named
formula has no available update. Treat that as satisfied when the command output
does not list the formula.

Failed when Homebrew is unavailable or the command errors unexpectedly.

## Apply

Use:

```sh
brew install <name>
brew upgrade <name>
```

Missing formulae are installed. Outdated formulae are upgraded.

Human `kitout apply` output prints a progress line before starting each install
or upgrade, for example `Upgrading formula go...`, because Homebrew can be slow
or quiet while it is working.

## Notes

Do not run `brew update` automatically in the MVP. That can be slow and surprising.

A future flag may support:

```sh
kitout apply --brew-update
```

## Implementation status

Implemented as `resources.BrewPackageResource`. Status and apply both use the
shared command runner interface so tests do not call real Homebrew.

## Shared expectations

Every resource must support:

- status check
- apply
- dry-run plan
- readable result messages
- unit tests

Status must never change the system.

Apply must be idempotent.
