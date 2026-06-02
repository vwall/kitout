# Product Brief

## Product name

Kitout

## Tagline

Equip a fresh Mac with your apps, packages, repos, dotfiles, and defaults.

## One-liner

Kitout is a Go CLI that reconciles your Mac against a declarative setup file.

## Problem

A developer's Mac setup is usually spread across:

- dotfiles
- Homebrew packages
- casks
- shell setup
- app preferences
- Git repositories
- local folders
- manual setup notes
- one-off scripts

This makes rebuilding a machine slow, error-prone, and hard to keep updated.

## Why existing tools are not quite enough

Pure dotfile managers are good at linking config files, but they do not fully represent the machine setup.

Shell scripts can set up a machine, but they are often imperative, fragile, and hard to safely re-run.

Heavy reproducibility tools can be powerful, but they may be more complex than needed for a personal Mac setup.

Kitout sits between those categories.

## Product promise

Kitout gives users a simple way to declare what a Mac should have and then check or apply that setup repeatedly.

## Example user story

As a developer, I want to run `kitout status` on my Mac and see what parts of my setup are missing, so I can bring the machine back to my expected state without guessing.

## Core flows

### Fresh Mac setup

1. Install Xcode Command Line Tools.
2. Install Kitout.
3. Clone setup repo.
4. Run `kitout doctor`.
5. Run `kitout apply --dry-run`.
6. Run `kitout apply`.

### Daily maintenance

1. Pull latest setup repo.
2. Run `kitout status`.
3. Run `kitout apply` if needed.

### Adding a new tool

1. Add package or cask to config.
2. Run `kitout status`.
3. Confirm it is missing.
4. Run `kitout apply`.
5. Commit the config change.

## Tone and personality

Kitout should feel:

- practical
- friendly
- calm
- safe
- confident

It should not feel:

- magical
- enterprise-heavy
- AI-branded
- overly clever
- destructive

## CLI personality

Use readable status lines:

```txt
✓ brew package git installed
✓ directory ~/code exists
✗ cask ghostty missing
! symlink ~/.zshrc points somewhere else

2 changes needed
Run: kitout apply --dry-run
```

## Product boundaries

Kitout is not a note-taking system for setup instructions. If something can be checked and applied, make it a resource. If it requires manual action, represent it as a manual task or doctor warning.
