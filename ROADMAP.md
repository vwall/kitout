# Roadmap

## Phase 0: Documentation and design

- Project charter
- Product brief
- Architecture overview
- Resource model spec
- CLI spec
- Config spec
- AGENTS.md
- Initial RFCs

## Phase 1: CLI skeleton

- Go module setup
- `kitout version`
- `kitout init`
- config loader
- config validation
- basic output renderer

## Phase 2: Engine and status

- resource interface
- planner
- status aggregation
- dry-run plan renderer
- JSON output foundation

## Phase 3: MVP resources

- brew packages
- asdf plugins and versions
- asdf `.tool-versions` entries
- casks
- directories
- symlinks
- Git repositories
- shell commands

## Phase 4: Safety and doctor

- doctor command
- path conflict detection
- symlink safety
- shell command confirmation
- better error messages

## Phase 5: Real Mac dogfood

- create private setup repo
- run on existing Mac
- test on fresh macOS user account
- refine config schema
- improve docs

## Phase 6: Distribution

- GitHub releases
- Homebrew tap
- release checksums
- install docs

## Later

- macOS defaults library
- config includes
- private config overlay
- Linux support
- templates
- secrets integration through external tools
- plugin API
