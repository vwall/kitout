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
```

Satisfied when the formula is installed.

Missing when Homebrew is available but the formula is not installed.

Failed when Homebrew is unavailable or the command errors unexpectedly.

## Apply

Use:

```sh
brew install <name>
```

## Notes

Do not run `brew update` automatically in the MVP. That can be slow and surprising.

A future flag may support:

```sh
kitout apply --brew-update
```

## Shared expectations

Every resource must support:

- status check
- apply
- dry-run plan
- readable result messages
- unit tests

Status must never change the system.

Apply must be idempotent.
