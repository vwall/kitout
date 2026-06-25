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

For commands that load config, explicit selection always wins:

```txt
--config PATH
```

Without `--config`, Kitout uses an implicit config only when the choice is
unambiguous:

- if only `./kitout.yaml` exists, use the repo-local config
- if only `~/.config/kitout/kitout.yaml` exists, use the home config
- if both exist, stop and ask the user to pass `--config`

When running from a private setup repo, `kitout status`, `kitout apply`, and
`kitout doctor` can use `./kitout.yaml` automatically if no home config exists:

```sh
kitout status
kitout apply --dry-run
```

If the home config also exists, select the setup repo explicitly:

```sh
kitout status --config ./kitout.yaml
kitout apply --config ./kitout.yaml --dry-run
```

This is intentionally boring: users should be able to tell which file is being
checked before Kitout reports or applies changes.

The loader and validator for this schema are implemented in `internal/config`.
`internal/resources.Build` converts validated config into ordered resources for
`kitout status`, `kitout apply --dry-run`, and `kitout apply`.

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

- `directories` is `[]string`
- `brew.taps` is `[]string`
- `brew.packages` is `[]string`
- `brew.casks` is `[]string`

Resources with named fields use typed structs:

- `asdf.plugins`
- `asdf.tool_versions`
- `repos`
- `copies`
- `symlinks`
- `symlink_groups`
- `macos_defaults`
- `login_shell`
- `shell`

Current resource implementation coverage:

- implemented resource packages: `brew.taps`, `brew.packages`, `brew.casks`,
  `asdf.plugins`, `asdf.tool_versions`, `directories`, `repos`, `copies`,
  `symlinks`, `symlink_groups`, `macos_defaults`, `login_shell`, and `shell`

## Compatibility

Top-level `casks` is still supported for schema version 1, but it is deprecated.
New configs should use `brew.casks`. Kitout reports a warning when top-level
`casks` contains entries, and validation rejects configs that set both
top-level `casks` and `brew.casks`.

## Required fields

The root `version` field is required.

Scalar resource entries must not be empty:

- `brew.taps[]`
- `brew.packages[]`
- `brew.casks[]`
- legacy `casks[]`
- `directories[]`

`brew.packages[]` entries may be simple formula names such as `git` or
fully-qualified tapped formula names such as `owner/repo/formula` when a tap
needs to be selected explicitly.

Named resources require the fields needed to identify and apply the resource:

- `asdf.plugins[].name`
- `asdf.plugins[].url`
- `asdf.plugins[].update_before_install` is optional
- `asdf.tool_versions[].path`
- `asdf.tool_versions[].tools`
- `repos[].path`
- `repos[].url`
- `copies[].source`
- `copies[].target`
- `copies[].replace` is optional
- `symlinks[].source`
- `symlinks[].target`
- `symlink_groups[].source_root`
- `symlink_groups[].target_root`
- `symlink_groups[].paths`
- `macos_defaults[].domain`
- `macos_defaults[].key`
- `macos_defaults[].type`
- `macos_defaults[].value`
- `login_shell.path`
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

`login_shell.path` must be either an absolute path or a `homebrew:<binary>`
resolver. `homebrew:fish` resolves to `$(brew --prefix)/bin/fish` during
resource status and apply checks. Arbitrary shell interpolation is rejected.

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
  taps:
    - vwall/kitout
  packages:
    - git
    - asdf
    - pnpm
    - gh
  casks:
    - ghostty
    - visual-studio-code
    - rectangle

asdf:
  plugins:
    - name: ruby
      url: https://github.com/asdf-vm/asdf-ruby.git
      update_before_install: true
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

directories:
  - ~/code
  - ~/.config
  - ~/.codex/skills

repos:
  - path: ~/code/example-project
    url: git@github.com:example/example-project.git
    branch: main

copies:
  - source: ./codex/skills/nuxt-practices
    target: ~/.codex/skills/nuxt-practices
    replace: false

symlinks:
  - source: ~/dotfiles/home/zshrc
    target: ~/.zshrc
    replace: false

symlink_groups:
  - source_root: ./home
    target_root: ~
    replace: false
    paths:
      - .zshrc
      - .gitconfig
      - .config/ghostty

macos_defaults:
  - domain: NSGlobalDomain
    key: AppleShowAllExtensions
    type: bool
    value: true

login_shell:
  path: homebrew:fish
  add_to_etc_shells: true

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

This matters for dotfile repos. If the selected config is
`~/code/setup/kitout.yaml`, then `./home/.zshrc` resolves to
`~/code/setup/home/.zshrc`, not the process working directory.

The implemented path-bearing resource fields are:

- `directories[]`
- `asdf.tool_versions[].path`
- `repos[].path`
- `copies[].source`
- `copies[].target`
- `symlinks[].source`
- `symlinks[].target`
- `symlink_groups[].source_root`
- `symlink_groups[].target_root`

`login_shell.path` is intentionally stricter. It accepts only absolute paths
and `homebrew:<binary>` resolvers so Kitout can resolve the path without
evaluating arbitrary shell text from YAML.

`symlink_groups[].paths[]` entries are relative path fragments below their
group roots. They are cleaned internally, but they do not resolve relative to
the config file directory on their own.

Example setup repo layout:

```txt
setup/
  kitout.yaml
  home/
    .zshrc
    .gitconfig
```

Config:

```yaml
symlink_groups:
  - source_root: ./home
    target_root: ~
    paths:
      - .zshrc
      - .gitconfig
```

Expanded targets:

```txt
~/code/setup/home/.zshrc    -> ~/.zshrc
~/code/setup/home/.gitconfig -> ~/.gitconfig
```

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

- same brew tap twice
- same brew package twice
- same asdf plugin name twice
- same asdf version twice within a plugin
- same asdf tool-versions path twice
- same cask twice
- same directory twice
- same copy target twice
- same target symlink twice
- same expanded symlink target twice, including conflicts between `symlinks`
  and `symlink_groups`
- same repository path twice
- same macOS default domain/key twice
- more than one login shell is not supported because `login_shell` is a single
  object
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
