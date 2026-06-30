# Cask Resource

## Purpose

The cask resource ensures a Homebrew cask application is installed.

## Config

```yaml
brew:
  taps:
    - homebrew/cask-fonts
  casks:
    - ghostty
    - visual-studio-code
    - rectangle
```

Top-level `casks` is not supported. Move cask entries under `brew.casks`.

Future expanded form:

```yaml
brew:
  casks:
    - name: ghostty
    - name: visual-studio-code
```

## Status check

Use:

```sh
brew list --cask --quiet
brew outdated --cask --quiet
```

Satisfied when the cask is installed.

Missing when Homebrew is available but the cask is not installed.

Still satisfied when the cask is installed and Homebrew reports that it is
outdated. Kitout reports an advisory such as `cask update available for ghostty`
with a `kitout upgrade cask:ghostty` or manual `brew upgrade --cask ghostty`
fix, but it does not treat the resource as drift from config.

Failed when Homebrew is unavailable or the command errors unexpectedly.

Kitout batches Homebrew cask installed and outdated checks for resources built
from the same config, so `kitout status` and `kitout apply --dry-run` inspect
each cask list once instead of running one `brew list` and one `brew outdated`
command per cask. During real apply execution, Kitout uses fresh uncached
resource checks before mutating so planning state cannot go stale.

## Apply

Use:

```sh
brew install --cask <name>
```

If a cask comes from a non-default tap, declare the tap under `brew.taps`.
Kitout applies Homebrew taps before casks.

Outdated casks are reported as advisories and are not upgraded by `kitout apply`.

## Upgrade

Use:

```sh
kitout upgrade --dry-run
kitout upgrade
kitout upgrade cask:ghostty
kitout upgrade --only cask
```

`kitout upgrade` checks the same managed `brew.casks` list, selects installed
casks with Homebrew outdated advisories, and runs:

```sh
brew upgrade --cask <name>
```

Missing casks are skipped with guidance to run `kitout apply` first. Casks that
are already current are reported unchanged. `kitout upgrade --dry-run` shows the
casks that would be upgraded without running `brew upgrade`. Pass a configured
resource ID such as `cask:ghostty` to upgrade only that cask.

## Safety

Cask installs may open macOS permission prompts or require app-specific post-install setup. Kitout should report success when Homebrew reports success, but it should not try to automate app login or private app configuration in the MVP.

## Implementation status

Implemented as `resources.CaskResource`. Status and apply both use the shared
command runner interface so tests do not call real Homebrew.

## Shared expectations

Every resource must support:

- status check
- apply
- dry-run plan
- readable result messages
- unit tests

Status must never change the system.

Apply must be idempotent.
