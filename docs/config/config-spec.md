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

The loader and validator for this schema are implemented in `internal/config`.
Resource packages consume these structs, but the CLI status command is still in
its config-validation scaffold and does not yet execute resource checks.

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

- `asdf.plugins`
- `asdf.tool_versions`
- `repos`
- `symlinks`
- `macos_defaults`
- `shell`

Current resource implementation coverage:

- implemented resource packages: `brew.packages`, `asdf.plugins`,
  `asdf.tool_versions`, `casks`, `directories`, `repos`, `symlinks`, and
  `shell`
- parsed and validated but not yet implemented as a resource package:
  `macos_defaults`

## Required fields

The root `version` field is required.

Scalar resource entries must not be empty:

- `brew.packages[]`
- `casks[]`
- `directories[]`

Named resources require the fields needed to identify and apply the resource:

- `asdf.plugins[].name`
- `asdf.plugins[].url`
- `asdf.tool_versions[].path`
- `asdf.tool_versions[].tools`
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

`asdf.plugins[].versions[]` and `asdf.tool_versions[].tools` values must be
exact versions. `latest` is rejected because it makes status mutable over time.

`shell[].when` is optional. When omitted, the shell command resource treats the
command as always needed. Supported conditions are:

```txt
always
missing-command:NAME
exists:PATH
missing:PATH
```

## Complete example

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
    - name: nodejs
      url: https://github.com/asdf-vm/asdf-nodejs.git
      versions:
        - 22.12.0
  tool_versions:
    - path: ~/.tool-versions
      tools:
        ruby: 3.3.6
        nodejs: 22.12.0

casks:
  - ghostty
  - visual-studio-code
  - 1password

directories:
  - ~/code
  - ~/.config

repos:
  - path: ~/code/aubs
    url: git@github.com:vrwaller/aubs.git
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

The config path passed to `--config` supports:

```txt
~
$HOME
absolute paths
```

Path-bearing resource fields support:

```txt
~
$HOME and other environment variables
relative paths from the config file directory
absolute paths
```

The implemented path-bearing resource fields are:

- `directories[]`
- `asdf.tool_versions[].path`
- `repos[].path`
- `symlinks[].source`
- `symlinks[].target`

Behavior:

- normalize paths internally
- display paths in user-friendly form when possible
- preserve exact config values in validation messages

## Environment variables

Support simple environment expansion for path fields.

Example:

```yaml
directories:
  - $HOME/code
```

Kitout does not support complex shell evaluation in config fields. Shell command
strings are explicit commands and are not expanded by the config loader.

## Duplicate detection

The validator should reject duplicate resources.

Examples:

- same brew package twice
- same asdf plugin name twice
- same asdf version twice within a plugin
- same asdf tool-versions path twice
- same cask twice
- same directory twice
- same target symlink twice
- same repository path twice
- same macOS default domain/key twice
- same shell command name twice

## Unknown fields

Unknown top-level fields fail validation.

Unknown resource fields fail validation.

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
