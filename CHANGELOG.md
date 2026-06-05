# Changelog

## 0.1.1 - Explicit config discovery

### Changed

- `kitout status`, `kitout apply`, and `kitout doctor` now require `--config` when both `./kitout.yaml` and `~/.config/kitout/kitout.yaml` exist.
- `./kitout.yaml` is still used automatically when it is the only implicit config file.

## 0.1.0 - Initial MVP

Kitout 0.1.0 is the first public release candidate for the macOS setup CLI.

### Added

- `kitout init` for creating a starter YAML config.
- `kitout status` for checking configured resources without changing the system.
- `kitout apply --dry-run` for previewing intended changes.
- `kitout apply` for applying missing or changed resources with confirmation for risky actions.
- `kitout doctor` for checking macOS prerequisites, Homebrew, Git, shell environment, config validity, and path permissions.
- `kitout version` with embedded build metadata for release artifacts.
- YAML config support for Homebrew packages, casks, asdf plugins and tool versions, directories, symlinks, symlink groups, Git repositories, macOS defaults, and explicit shell commands.
- Human-readable and JSON output for status, apply planning, and doctor checks.
- macOS release automation for Apple Silicon and Intel binaries with checksums.

### Notes

- The first release is macOS-focused.
- Homebrew installation is checked but never installed automatically.
- Shell commands run only when explicitly listed in config and confirmed, unless `--yes` is passed.
- Config includes, templates, secrets, plugins, Linux support, and Windows support are outside the 0.1.0 scope.
