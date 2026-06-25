# Brew Resources

## Purpose

The brew tap resource ensures a Homebrew tap is available. The brew package
resource ensures a Homebrew formula is installed. The cask resource ensures a
Homebrew cask application is installed.

## Config

```yaml
brew:
  taps:
    - vwall/kitout
  packages:
    - git
    - vwall/kitout/kitout
    - ruby
    - node
  casks:
    - ghostty
    - visual-studio-code
```

Future expanded form:

```yaml
brew:
  taps:
    - name: vwall/kitout
  packages:
    - name: git
    - name: ruby
      version: latest
```

## Tap status check

Use:

```sh
brew tap
```

Satisfied when the tap appears in Homebrew's tapped repository list.

Missing when Homebrew is available but the tap is not listed.

Failed when Homebrew is unavailable or the command errors unexpectedly.

Kitout batches tap list checks for resources built from the same config, so
`kitout status` and `kitout apply --dry-run` inspect the tap list once instead
of running one `brew tap` command per tap. During real apply execution, Kitout
uses fresh uncached resource checks before mutating so planning state cannot go
stale.

## Package status check

Use:

```sh
brew list --formula --quiet
brew outdated --formula --quiet
```

Satisfied when the formula is installed.

Missing when Homebrew is available but the formula is not installed.

Changed when the formula is installed and Homebrew reports that it is outdated.
Human status output marks this as `changed`.

Kitout batches Homebrew installed and outdated checks for resources built from
the same config, so `kitout status` and `kitout apply --dry-run` inspect each
formula list once instead of running one `brew list` and one `brew outdated`
command per package. During real apply execution, Kitout uses fresh uncached
resource checks before mutating so planning state cannot go stale.

Some Homebrew versions return exit code 1 from `brew outdated` when no formula
has an available update. Treat that as satisfied when the command output does
not list the formula.

Failed when Homebrew is unavailable or the command errors unexpectedly.

## Apply

Use:

```sh
brew tap <name>
brew install <name>
brew upgrade <name>
```

Missing taps are added before formulae are installed.

Missing formulae are installed. Outdated formulae are upgraded.

Use fully-qualified formula names such as `owner/repo/formula` when a tapped
formula needs to be selected explicitly.

Human `kitout apply` output prints a progress line before starting each tap
addition, install, or upgrade, for example `Adding Homebrew tap vwall/kitout...`
or `Upgrading formula go...`, because Homebrew can be slow or quiet while it is
working.

## Notes

Do not run `brew update` automatically in the MVP. That can be slow and surprising.

A future flag may support:

```sh
kitout apply --brew-update
```

## Implementation status

Implemented as `resources.BrewTapResource` and `resources.BrewPackageResource`.
Status and apply both use the shared command runner interface so tests do not
call real Homebrew.

## Shared expectations

Every resource must support:

- status check
- apply
- dry-run plan
- readable result messages
- unit tests

Status must never change the system.

Apply must be idempotent.
