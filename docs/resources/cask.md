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

Future expanded form:

```yaml
casks:
  - name: ghostty
  - name: visual-studio-code
```

## Status check

Use:

```sh
brew list --cask --quiet
```

Satisfied when the cask is installed.

Missing when Homebrew is available but the cask is not installed.

Failed when Homebrew is unavailable or the command errors unexpectedly.

Kitout batches Homebrew cask installed checks for resources built from the same
config, so `kitout status` and `kitout apply --dry-run` inspect the cask list
once instead of running one `brew list` command per cask. During real apply
execution, Kitout uses fresh uncached resource checks before mutating so planning
state cannot go stale.

## Apply

Use:

```sh
brew install --cask <name>
```

If a cask comes from a non-default tap, declare the tap under `brew.taps`.
Kitout applies Homebrew taps before casks.

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
