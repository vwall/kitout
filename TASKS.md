# Tasks

This checklist tracks the next small slices of Kitout work. Keep `ROADMAP.md`
as the phase-level map; use this file for near-term implementation tasks.

## Now: Phase 1 CLI skeleton

- [x] Set up Go module and command entrypoint.
- [x] Add `kitout version`.
- [x] Add `kitout init`.
- [x] Add `internal/config` package with typed YAML structs.
- [x] Add config path resolution for explicit `--config`.
- [x] Add config loading from disk.
- [x] Add config validation with clear, structured errors.
- [x] Add a basic human-readable output renderer.
- [x] Decide how much of `--json` should be scaffolded in Phase 1.

## Next: first status path

- [x] Add `kitout status` command stub.
- [x] Wire `kitout status` to load and validate config.
- [x] Render config validation failures with actionable messages.
- [x] Return planned exit codes for OK, validation, and runtime errors.
- [x] Add tests for status with valid, missing, and invalid config files.

## Next: Phase 2 engine foundation

- [x] Add `internal/engine/resource.go` with the `Resource` interface.
- [x] Add status and apply result types.
- [x] Add explicit status values: satisfied, missing, changed, failed, skipped.
- [x] Add planner skeleton that checks resources without mutation.
- [x] Add dry-run plan result shape.
- [x] Add tests for planner aggregation and failure handling.

## Then: MVP resource scaffolding

- [x] Add command runner interface in `internal/platform`.
- [x] Add directory resource with status, apply, and dry-run tests.
- [x] Add symlink resource with safety checks before overwrite support.
- [x] Add brew package resource using the command runner interface.
- [x] Add asdf plugin and `.tool-versions` resources using the command runner interface.
- [x] Add cask resource using the command runner interface.
- [x] Add Git repo resource using the command runner interface.
- [x] Add shell command resource with explicit config opt-in.

## Then: CLI resource execution

- [x] Add config-to-resource builder with stable execution order.
- [x] Wire `kitout status` to resource checks and plan rendering.
- [x] Add sequential executor for planned apply actions.
- [x] Add `kitout apply --dry-run`.
- [x] Add `kitout apply`.
- [x] Add JSON plan output for status and dry-run.

## Documentation follow-up

- [x] Update `docs/config/config-spec.md` when config structs become executable.
- [x] Update `docs/cli/cli-spec.md` when command behavior changes.
- [x] Keep `examples/kitout.yaml` aligned with loader validation.
- [x] Add short notes to resource docs as each resource becomes implemented.

## Next: Phase 4 safety and doctor

- [x] Add `kitout doctor` command with non-mutating prerequisite checks.
- [x] Check macOS, CPU architecture, Xcode Command Line Tools, Homebrew, Git, and config validity.
- [x] Add human-readable and JSON doctor output.
- [x] Add path permission checks for configured write targets.
- [x] Add Homebrew path checks for Apple Silicon.
- [x] Add shell environment checks.
- [x] Add confirmation flow for risky apply actions.

## Next: dogfood ergonomics

- [x] Add `symlink_groups` config support for dotfile-style bulk symlinks.
- [x] Expand grouped symlink entries into normal symlink resources during config-to-resource building.
- [x] Validate grouped symlink roots, paths, and duplicate expanded targets.
- [x] Document grouped symlink examples in README and examples when implemented.

## Next: Phase 5 real Mac dogfood

- [x] Create a private Kitout config repo for personal machine setup.
- [x] Add a representative real-world `kitout.yaml`.
- [x] Run `kitout status` against the private config on an existing Mac.
- [x] Run `kitout apply --dry-run` and review planned changes for safety and noise.
- [x] Apply a small subset of low-risk resources.
- [x] Capture gaps in config schema, output clarity, and resource behavior.
- [x] Test `kitout doctor` on the dogfood machine.
- [x] Test against a fresh macOS user account or clean VM.
- [x] Refine docs based on real setup friction.
- [x] Decide what must be fixed before distribution work begins.

## Next: launch and distribution readiness

- [x] Decide whether the first launch target is private beta, public `0.1.0`,
      or website-only preview.
- [x] Add release build automation for macOS `arm64` and `amd64` binaries.
- [x] Ensure release binaries embed version, commit, and build date metadata.
- [x] Generate checksums from the exact release artifacts that will be
      published.
- [x] Prepare a short changelog for the first tagged release.
- [x] Create the GitHub release flow for publishing binaries and checksums.
- [x] Prepare a Homebrew tap formula template for `kitout`.
- [x] Update launch-facing docs so install commands match the actual release
      path.
- [x] Remove or mark unsupported CLI promises, including the optional
      `kitout list` command in the CLI spec.
- [x] Add a real public security contact and disclosure policy.
- [x] Run a trademark and package registry review for the Kitout name.
- [x] Decide whether to keep the new landing page HTML, generated assets,
      screenshots, and design QA notes in the repo.
- [x] Remove local-only artifacts such as `.DS_Store` before staging launch
      assets.
- [x] Update `MANIFEST.md` if launch assets, scripts, or new RFCs are kept.
- [x] Run `go test ./...`, `go vet ./...`, and `make smoke-distribution`
      locally with a writable Go module cache.
- [x] Run `go test ./...`, `go vet ./...`, and `make smoke-distribution`
      from a clean temporary copy with writable Go caches.
- [x] Resolve release-environment warning guidance, including Go module cache
      permission noise.
- [x] Re-run `go test ./...`, `go vet ./...`, and `make smoke-distribution`
      from a clean checkout immediately before tagging.
- [ ] Tag the first release after docs and smoke checks are verified.
- [ ] Publish the Homebrew tap formula after release checksums exist.
- [ ] Verify the Homebrew install path from a clean machine or disposable macOS
      user account.
