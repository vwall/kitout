# macOS Defaults Resource

## Purpose

The macOS defaults resource ensures selected macOS preference values are set.

## Config

```yaml
macos_defaults:
  - domain: NSGlobalDomain
    key: AppleShowAllExtensions
    type: bool
    value: true
```

## Supported types

MVP types:

```txt
bool
int
float
string
```

## Status check

Use:

```sh
defaults read <domain> <key>
```

Compare the actual value to the expected value.

## Apply

Use:

```sh
defaults write <domain> <key> -bool true
```

or the appropriate type flag.

## Safety

macOS defaults can be fragile. Start with a small set of tested defaults.

Do not include a large library of defaults in the MVP.

Each example default should have a comment explaining what it changes.

## Restart behavior

Some defaults require restarting apps, Finder, Dock, or SystemUIServer.

For the MVP, report a note rather than killing processes automatically.

Future config may support:

```yaml
restart:
  - Finder
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
