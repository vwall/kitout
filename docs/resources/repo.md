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

## Non-goals

Do not auto-pull repositories in the MVP.

Do not overwrite local changes.

Do not switch branches automatically if the repo already exists.

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
