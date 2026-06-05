# Security

## Core safety principle

Kitout runs on a user's local machine and can modify important files. Safety is a product requirement, not an add-on.

## Secrets

Do not store secrets in Kitout config.

Examples of secrets:

- API keys
- private SSH keys
- tokens
- passwords
- `.env` values

Kitout may install or check secret-management tools, but it should not become one.

## Shell commands

Shell commands are allowed only when explicitly listed in config.

Kitout must show shell commands during dry-run.

Kitout should require confirmation before running shell commands unless `--yes` is passed.

## File safety

Kitout must not overwrite existing files by default.

Symlink replacement must be explicit.

Potential future backup behavior must be opt-in and well documented.

## Reporting vulnerabilities

Report vulnerabilities by creating a GitHub issue.

Preferred channels:

- GitHub private vulnerability reporting for `github.com/vwall/kitout`, when enabled
- GitHub issues for `github.com/vwall/kitout`

Please include:

- affected Kitout version or commit
- config/resource type involved
- steps to reproduce
- expected impact
- any suggested mitigation

The maintainer will acknowledge reports as soon as practical, triage the issue,
and coordinate a fix before public disclosure when the report describes a real
security impact.
