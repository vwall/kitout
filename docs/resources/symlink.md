# Symlink Resource

## Purpose

The symlink resource ensures a target path points to a source path.

## Config

```yaml
symlinks:
  - source: ~/dotfiles/home/zshrc
    target: ~/.zshrc
    replace: false
```

## Grouped config

Managing many dotfile links one-by-one can become repetitive.
`symlink_groups` allows shared source and target roots while still expanding to
independent symlink resources internally.

Shape:

```yaml
symlink_groups:
  - source_root: ./home
    target_root: ~
    replace: false
    paths:
      - .zshrc
      - .gitconfig
      - .config/ghostty
      - .config/nvim
```

Each path would expand as if it had been written explicitly:

```txt
./home/.zshrc          -> ~/.zshrc
./home/.gitconfig      -> ~/.gitconfig
./home/.config/ghostty -> ~/.config/ghostty
./home/.config/nvim    -> ~/.config/nvim
```

Grouped symlinks should use the same status, apply, dry-run, duplicate-target,
and replacement safety rules as explicit `symlinks` entries.

`paths` entries must be relative paths below the group roots. Absolute paths,
`.` entries, and paths that escape with `..` are rejected during validation.

For a setup repo, prefer repo-relative roots such as `source_root: ./home`.
Relative path-bearing fields resolve from the directory containing the selected
config file. Home-relative roots such as `~/dotfiles/home` are valid only when
that directory exists on the target machine.

## Status check

Satisfied when:

- target exists
- target is a symlink
- target points to source

Missing when target does not exist.

Changed when target exists but points somewhere else.

Failed when target exists as a normal file or directory and replacement is not allowed.

## Apply

If target is missing:

```sh
ln -s <source> <target>
```

If target exists and `replace: true`, replace it safely.

If target exists and `replace: false`, fail with guidance.

## Safety

Default to `replace: false`.

Do not overwrite a real file unless the config explicitly allows it.

`kitout apply` also asks for confirmation before replacing an existing symlink
target unless `--yes` is passed. Creating a missing symlink does not require
confirmation.

When dogfooding on an existing Mac, expect dotfile targets such as `~/.zshrc`,
`~/.gitconfig`, or app config directories to already exist. First run
`kitout apply --dry-run`, move or back up files you want Kitout to own, and set
`replace: true` only for targets where replacement is intentional.

A future backup option may support:

```yaml
backup: true
```

## Implementation status

Implemented as `resources.SymlinkResource`. Status uses `os.Lstat` so it can
inspect links without following them, and apply only replaces an existing target
when `replace: true` is configured. CLI apply confirmation is required for
replacement unless `--yes` is passed.

`symlink_groups` is implemented in the config layer and expands into ordinary
`resources.SymlinkResource` entries during config-to-resource building. Grouped
targets also participate in duplicate-target validation and doctor path
permission checks.

## Shared expectations

Every resource must support:

- status check
- apply
- dry-run plan
- readable result messages
- unit tests

Status must never change the system.

Apply must be idempotent.
