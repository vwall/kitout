# Cask Resource

## Purpose

The cask resource ensures a Homebrew cask application is installed.

## Config

```yaml
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
brew list --cask <name>
```

Satisfied when the cask is installed.

Missing when Homebrew is available but the cask is not installed.

Failed when Homebrew is unavailable or the command errors unexpectedly.

## Apply

Use:

```sh
brew install --cask <name>
```

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
