# Changelog

## 1.1.0 - Copy and login shell resources

Kitout 1.1.0 adds optional resource support for file/directory copies and the
current user's macOS login shell.

### Added

- Added `copies` config support for managing physical file and directory copies.
- Added `login_shell` config support for managing the current user's macOS login
  shell.
- Added dedicated resource documentation for copies and login shell management.
- Added CLI rendering, dry-run messages, doctor write-target checks, and tests
  for the new resource types.

### Safety

- Copy replacements require `replace: true` and are confirmation-gated unless
  `--yes` is passed.
- Login-shell changes are confirmation-gated unless `--yes` is passed.
- Copy sources reject symlinks and unsupported file types.

### Distribution

- Prepared the GitHub release, install docs, and Homebrew formula template for
  `v1.1.0` macOS arm64 and amd64 artifacts.

### Notes

- This is an additive `version: 1` schema release. Existing 1.0.0 configs do
  not need to change.

## 1.0.0 - Stable Mac setup CLI

Kitout 1.0.0 is the first stable public release for declarative macOS setup.

### Added

- Stable YAML config schema, `version: 1`.
- Homebrew tap distribution target.
- Release-note preparation tooling and tests.
- GitHub Pages documentation site assets and publishing workflow.

### Changed

- `kitout status`, `kitout apply`, and `kitout doctor` now require `--config` when both `./kitout.yaml` and `~/.config/kitout/kitout.yaml` exist.
- `./kitout.yaml` is still used automatically when it is the only implicit config file.
- Added `--color` and `--no-color` output controls.

### Fixed

- Normalized `TMPDIR` handling in the distribution smoke test.

### Notes

- This release includes the prepared but unpublished `0.1.2` changes.
- Config schema `version: 1` is now considered stable.

## 0.1.1 - Local config discovery

### Changed

- `kitout status`, `kitout apply`, and `kitout doctor` now select `./kitout.yaml` before `~/.config/kitout/kitout.yaml` when `--config` is not passed.

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
