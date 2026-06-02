# Config Specification

## Format

Kitout uses YAML for the MVP.

Default path:

```txt
~/.config/kitout/kitout.yaml
```

Repo-local path:

```txt
./kitout.yaml
```

Use `--config` to select a specific config file.

## Version

All config files should include a version.

```yaml
version: 1
```

The Go representation lives in `internal/config`. The MVP schema version is
`config.CurrentVersion`, currently `1`.

Validation should reject a missing `version` field and any version other than
`config.CurrentVersion`.

Scalar resource lists remain scalar in Go:

- `casks` is `[]string`
- `directories` is `[]string`
- `brew.packages` is `[]string`

Resources with named fields use typed structs:

- `repos`
- `symlinks`
- `macos_defaults`
- `shell`

## Required fields

The root `version` field is required.

Scalar resource entries must not be empty:

- `brew.packages[]`
- `casks[]`
- `directories[]`

Named resources require the fields needed to identify and apply the resource:

- `repos[].path`
- `repos[].url`
- `symlinks[].source`
- `symlinks[].target`
- `macos_defaults[].domain`
- `macos_defaults[].key`
- `macos_defaults[].type`
- `macos_defaults[].value`
- `shell[].name`
- `shell[].command`

`macos_defaults[].type` must be one of:

```txt
bool
int
float
string
```

## Complete example

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
  - rectangle

directories:
  - ~/code
  - ~/.config

repos:
  - path: ~/code/example-project
    url: git@github.com:example/example-project.git
    branch: main

symlinks:
  - source: ~/dotfiles/home/zshrc
    target: ~/.zshrc
    replace: false

macos_defaults:
  - domain: NSGlobalDomain
    key: AppleShowAllExtensions
    type: bool
    value: true

shell:
  - name: Enable Corepack
    command: corepack enable
    when: missing-command:pnpm
```

## Path expansion

Kitout should support:

```txt
~
$HOME
relative paths from the config file directory
absolute paths
```

Recommended behavior:

- normalize paths internally
- display paths in user-friendly form when possible
- preserve exact config values in validation messages

## Environment variables

Support simple environment expansion for paths and commands.

Example:

```yaml
directories:
  - $HOME/code
```

Do not support complex shell evaluation in config fields.

## Duplicate detection

The validator should reject duplicate resources.

Examples:

- same brew package twice
- same cask twice
- same directory twice
- same target symlink twice
- same repository path twice
- same macOS default domain/key twice
- same shell command name twice

## Unknown fields

Unknown top-level fields should fail validation.

Unknown resource fields should fail validation.

This keeps config mistakes visible.

## Comments

YAML comments are allowed.

Kitout does not need to preserve comments when reading config unless a future `kitout fmt` command is added.

## Private config

Users may keep a private override file outside Git.

Recommended future path:

```txt
~/.config/kitout/private.yaml
```

Do not implement merging in the MVP unless needed.

## Secrets

Do not put secrets in Kitout config.

Allowed:

```yaml
shell:
  - name: Check 1Password CLI
    command: op --version
```

Not allowed:

```yaml
secrets:
  github_token: ghp_example
```

## Validation errors

Validation errors should be specific.

Bad:

```txt
invalid config
```

Good:

```txt
Invalid config: symlinks[0].target is required
```

## Schema stability

The first stable public version should use:

```yaml
version: 1
```

Breaking schema changes should require a new version.
