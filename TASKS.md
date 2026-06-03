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
- [ ] Add symlink resource with safety checks before overwrite support.
- [ ] Add brew package resource using the command runner interface.
- [ ] Add cask resource using the command runner interface.
- [ ] Add Git repo resource using the command runner interface.
- [ ] Add shell command resource with explicit config opt-in.

## Documentation follow-up

- [ ] Update `docs/config/config-spec.md` when config structs become executable.
- [ ] Update `docs/cli/cli-spec.md` when command behavior changes.
- [ ] Keep `examples/kitout.yaml` aligned with loader validation.
- [ ] Add short notes to resource docs as each resource becomes implemented.
