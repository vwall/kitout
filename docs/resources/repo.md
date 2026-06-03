# Git Repository Resource

## Purpose

The repo resource ensures a Git repository exists at a local path.

## Config

```yaml
repos:
  - path: ~/code/example-project
    url: git@github.com:example/example-project.git
    branch: main
```

## Status check

Satisfied when:

- path exists
- path is a Git repository
- remote origin matches the configured URL when URL is provided

Missing when the path does not exist.

Changed when path exists but is not the expected repo.

## Apply

If path is missing:

```sh
git clone <url> <path>
```

If branch is set:

```sh
git clone --branch <branch> <url> <path>
```

For private repositories, make sure Git can authenticate before the first apply.
SSH URLs need a usable key and host trust; HTTPS URLs may need an existing
credential helper session. Kitout reports Git failures, but it does not
configure authentication in the MVP.

## Non-goals

Do not auto-pull repositories in the MVP.

Do not overwrite local changes.

Do not switch branches automatically if the repo already exists.

## Implementation status

Implemented as `resources.RepoResource`. Status checks the local path, verifies
that Git sees a work tree, and compares `origin` to the configured URL through
the shared command runner interface.

## Future options

```yaml
repos:
  - path: ~/code/example-project
    url: git@github.com:example/example-project.git
    branch: main
    pull: false
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
