# Login Shell Resource

## Purpose

The login shell resource ensures the current user's macOS login shell matches
the configured shell path.

## Config

```yaml
brew:
  packages:
    - fish

login_shell:
  path: homebrew:fish
  add_to_etc_shells: true
```

`homebrew:fish` resolves to `$(brew --prefix)/bin/fish` during status and apply
without evaluating arbitrary shell text from the config.

An explicit absolute path is also valid:

```yaml
login_shell:
  path: /opt/homebrew/bin/fish
  add_to_etc_shells: true
```

## Status Check

Satisfied when:

- the resolved shell binary exists and is executable
- the resolved path is listed in `/etc/shells`
- the current user's login shell already matches the resolved path

Missing when:

- the resolved shell binary is missing
- the resolved path is not listed in `/etc/shells` and
  `add_to_etc_shells: true`

Changed when:

- the shell is allowed, but the current user's login shell differs

Failed when:

- Homebrew path resolution fails
- `/etc/shells` cannot be read
- the current user's shell cannot be inspected
- the shell is not listed in `/etc/shells` and `add_to_etc_shells: false`

## Apply

When `add_to_etc_shells: true`, apply appends the resolved shell path to
`/etc/shells` if needed.

When the current user's login shell differs, apply runs:

```sh
chsh -s <resolved-shell-path>
```

Apply is idempotent and does nothing when the resource is already satisfied.

## Safety

Changing the login shell is a risky action. `kitout apply` requires
confirmation unless `--yes` is passed.

Kitout does not install Homebrew or fish from this resource. Pair it with an
explicit Homebrew package when the shell should be installed:

```yaml
brew:
  packages:
    - fish
```

Kitout does not support arbitrary shell interpolation such as:

```yaml
login_shell:
  path: "$(brew --prefix)/bin/fish"
```

Use `homebrew:fish` instead.
