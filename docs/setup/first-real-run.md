# First Real Run

This guide captures the rough edges that show up when Kitout is pointed at a
real personal setup repo for the first time.

## Pick the config explicitly

Kitout defaults to:

```txt
~/.config/kitout/kitout.yaml
```

When dogfooding from a cloned setup repo, pass `--config` until you intentionally
install or copy the config to the default path:

```sh
kitout doctor --config ~/code/setup/kitout.yaml
kitout status --config ~/code/setup/kitout.yaml
kitout apply --config ~/code/setup/kitout.yaml --dry-run
```

Do the same with `go run` when working from source:

```sh
go run ./cmd/kitout doctor --config ~/code/setup/kitout.yaml
go run ./cmd/kitout status --config ~/code/setup/kitout.yaml
go run ./cmd/kitout apply --config ~/code/setup/kitout.yaml --dry-run
```

`./kitout.yaml` is useful for a setup repo, but it is not auto-selected by the
current CLI. Using `--config` keeps accidental runs against the wrong file
visible.

## Run doctor before apply

Use `doctor` with the same config you intend to apply:

```sh
kitout doctor --config ~/code/setup/kitout.yaml
```

This catches common first-run blockers before the apply plan:

- Xcode Command Line Tools are missing
- Homebrew or Git is unavailable
- `/opt/homebrew/bin` is missing from `PATH` on Apple Silicon
- the selected config is invalid
- configured write targets are not writable

Fix failed doctor items before applying. Warnings are not always blockers, but
they usually explain noisy status or dry-run output.

## Expect prerequisites to affect the first plan

Kitout builds a plan by checking all resources before it applies any of them.
That means dependent resources can fail during the first dry-run even if an
earlier resource would install the dependency.

Examples:

- `brew.packages` can include `asdf`, but `asdf.plugins` still need `asdf` to be
  available during the planning check.
- repo resources that use SSH URLs need Git and working SSH authentication before
  status can fully inspect or clone them.

`kitout apply` stops before changing anything when the plan contains failed or
unknown resources. On a fresh machine, use one of these small-pass approaches:

- install the prerequisite manually, then rerun `doctor` and `status`
- temporarily comment out dependent sections until the prerequisite exists
- keep a small bootstrap config for Homebrew, Git, asdf, and base directories

Example bootstrap pass:

```sh
kitout apply --config ~/code/setup/bootstrap.kitout.yaml --dry-run
kitout apply --config ~/code/setup/bootstrap.kitout.yaml
kitout doctor --config ~/code/setup/kitout.yaml
kitout apply --config ~/code/setup/kitout.yaml --dry-run
```

For very new machines, get prerequisites such as Homebrew, Git, and asdf onto
`PATH`, then rerun status once those tools are visible.

## Use repo-relative dotfile paths intentionally

Path fields such as symlink sources, symlink group roots, directories, repos, and
`.tool-versions` paths support relative paths. Relative paths resolve from the
directory containing the selected config file.

For a setup repo like:

```txt
setup/
  kitout.yaml
  home/
    .zshrc
    .gitconfig
```

prefer:

```yaml
symlink_groups:
  - source_root: ./home
    target_root: ~
    replace: false
    paths:
      - .zshrc
      - .gitconfig
```

Using `~/dotfiles/home` is also valid, but only if that directory really exists
at that home-relative path on the target machine.

## Treat existing files as decisions

Symlinks default to `replace: false`. If a target like `~/.zshrc` already exists
as a regular file or points somewhere else, Kitout reports the conflict instead
of overwriting it silently.

The safest first-run loop is:

1. Run `kitout apply --dry-run`.
2. Move or back up any real files you want Kitout to manage.
3. Set `replace: true` only for targets you are comfortable replacing.
4. Run `kitout apply` and answer the confirmation prompt for risky changes.

Use `--yes` only after the dry-run output is familiar.

## Keep shell resources rare

Shell resources are useful for setup gaps, but they always deserve extra review.
They run through `sh -c`, appear in dry-run output, and require confirmation
unless `--yes` is passed.

Prefer a first-class resource when one exists. Keep shell commands named,
specific, and idempotent.
