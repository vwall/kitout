# Kitout Project Charter

## Purpose

Kitout exists to make a fresh Mac feel like the user's machine again.

It does this by reading a declarative configuration file and reconciling the machine toward that desired state.

Kitout manages more than dotfiles. It can manage packages, apps, folders, repositories, symlinks, and selected system defaults.

## Vision

A developer should be able to clone a private setup repo, install Kitout, run one command, and see exactly what is already configured and what still needs to be applied.

## Inspiration

Kitout is inspired by Bork's concept of declarative assertions for system state. The project should borrow the philosophy, not the shell implementation.

## Goals

1. Provide a clear declarative config format.
2. Make machine setup repeatable.
3. Keep dry-run and status safe.
4. Support first-class macOS setup.
5. Make every resource idempotent.
6. Keep the tool easy for Codex and humans to modify.
7. Make failure states readable and actionable.

## Non-goals

1. Strict Bork compatibility.
2. Parsing arbitrary Bash config files.
3. Replacing Homebrew.
4. Replacing dedicated secret managers.
5. Becoming a full configuration-management platform.
6. Becoming a Nix alternative.
7. Managing production servers.

## Target users

The first target user is a developer who wants to rebuild a personal Mac setup.

Secondary users include:

- startup engineers with repeatable laptop setup needs
- freelancers who switch machines
- developers who want a lighter alternative to Nix
- developers who want more than dotfile symlinks

## Product principles

### Be clear

The user should always understand what Kitout is checking, what it plans to change, and what happened.

### Be safe

Status and dry-run must never mutate the system.

### Be boring

Prefer explicit config and predictable behavior over clever magic.

### Be local-first

A local config file should be enough for the MVP.

### Be Mac-first

The first version should deeply support modern macOS setups before expanding to other platforms.

## MVP definition

The MVP is complete when Kitout can:

- read `kitout.yaml`
- validate config
- check status for resources
- dry-run an apply
- apply Homebrew packages
- install asdf plugins and exact runtime versions
- update explicit `.tool-versions` entries
- apply Homebrew casks
- create directories
- create symlinks
- clone Git repositories
- run approved shell commands
- print a readable summary
- provide a useful `doctor` command

## Success criteria

The first real test is whether the author can use Kitout to set up their own Mac from a private setup repo.

The second test is whether a new contributor can understand the resource model and add a new resource without changing the whole engine.
