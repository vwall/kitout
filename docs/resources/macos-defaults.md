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

Implemented as `resources.MacOSDefaultResource`. Status uses the shared command
runner to call:

```sh
defaults read <domain> <key>
```

When the value matches, `defaults read-type <domain> <key>` verifies its stored
type. String whitespace is significant. Missing keys are reported as `missing`;
existing keys with different values or types are reported as `changed`.

## Apply

Apply first checks status, then uses the shared command runner to call:

```sh
defaults write <domain> <key> -bool true
```

or the appropriate type flag:

```txt
-bool
-int
-float
-string
```

Apply is idempotent. If the value is already set, no write command runs.

## Safety

macOS defaults can be fragile. Start with a small set of tested defaults.

Do not include a large library of defaults in the MVP.

Each example default should have a comment explaining what it changes.

## Restart behavior

Some defaults require restarting apps, Finder, Dock, or SystemUIServer.

Kitout does not restart processes automatically. Users should restart affected
apps or services when a default requires it.

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
