# CLI Specification

## Command name

```sh
kitout
```

## Global flags

```txt
--config PATH       Path to config file (required when both ./kitout.yaml and home config exist)
--verbose           Show status progress and stream subprocess output during apply or upgrade
--quiet             Reduce output
--color             Force colored output
--no-color          Disable colored output
--json              Print machine-readable JSON output
--yes               Bypass apply confirmations for risky changes
```

These flags are currently parsed by the root command and by implemented
subcommands. `doctor`, `status`, `apply`, and `upgrade` use `--config` when it is passed.
Without `--config`, they use `./kitout.yaml` only when no home config exists,
use `~/.config/kitout/kitout.yaml` only when no local config exists, and fail
with guidance when both files exist. `init` writes `./kitout.yaml` in the
current directory by default; pass `--home` to create the home config instead.
Human
output colors status, progress, and action markers when stdout is an interactive terminal;
redirected output remains plain text by default, `--color` forces ANSI color,
and `--no-color` disables ANSI color markers. Human output includes both
symbols and text labels so status remains readable without color. `status`,
`apply`, `upgrade`, and `doctor` print a short startup line before slower checks begin so
command startup does not look stuck. `status --verbose` prints a progress line
before each resource status check. For batched Homebrew status checks, default
`status` output prints one coarse tap-list line, one package-list line, and one
cask-list line instead of per-resource `Checking ...` lines. `apply` prints a
progress line before each non-Homebrew resource status check and before each
resource apply starts. For batched Homebrew status checks, default `apply`
output prints one coarse inspection line for taps, one for formulae, and one for
casks instead of per-resource `Checking ...` lines. `upgrade` uses the same
batched Homebrew formula and cask inspection style before selecting outdated
managed items. During human `kitout apply` and `kitout upgrade` runs,
`--verbose` renders each resource status check, renders each subprocess command
before it starts, and streams that subprocess stdout and stderr as it runs.
`--verbose` does not stream output for `--dry-run`, `--json`, or `--quiet`
runs. `--json` and `--quiet` currently affect `context`, `status`, `apply`,
`upgrade`, `doctor`, and `explain`.

For mutation commands whose output is only diagnostic, failure reports retain
the final 64 KiB of each output stream and explicitly mark truncated output.
Use `--verbose` to stream complete logs. Commands whose output Kitout parses,
including asdf installs and public-key extraction, retain complete capture.

When running from a private setup repo that contains `kitout.yaml`, pass
`--config ./kitout.yaml` if a home config also exists. Kitout prints the selected
path before running checks.

Config deprecation warnings are non-fatal. Human `status`, `apply`, and
`upgrade` render them once on stderr after the selected config is loaded. JSON
output includes them under `config.warnings`. `doctor` reports a deprecated but
otherwise valid config as a `warn` Config item.

Implicit dependency checks and first-class resources resolve external tools such
as `brew`, `git`, and `xcode-select` from trusted system locations instead of
the caller's ambient `PATH`. Absolute executable paths used by implicit checks
must also be under trusted system or Homebrew command roots. Explicit `shell`
resources still run through the user PATH because those commands are
intentionally listed in config and require apply confirmation.

## Commands

### `kitout init`

Creates a starter config file and, when requested, repo-local agent guidance.

```sh
kitout init
kitout init --home
kitout init --agents
kitout init --no-agents-warning
kitout init --config ./kitout.yaml
kitout init --config ~/.config/kitout/kitout.yaml
kitout init --config ./kitout.yaml --agents
kitout init --config ./kitout.yaml --no-agents-warning
```

Behavior:

- without `--config`, create or use `./kitout.yaml` in the current directory
- with `--home`, create or use `~/.config/kitout/kitout.yaml`
- reject combining `--home` and `--config`
- create parent config directory if missing
- refuse to overwrite existing config unless `--force` is passed
- write a starter config that validates and can be checked by `status` without
  manual edits
- with `--agents`, create or update `AGENTS.md` with compact Kitout guidance
  for coding agents
- with `--no-agents-warning`, create or update
  `.kitout/agent-guidance.yaml` so `doctor` does not warn about a missing
  repo-local `AGENTS.md`
- when `--agents` is passed and the config already exists, leave the config
  unchanged and still create or update `AGENTS.md`
- when `--no-agents-warning` is passed and the config already exists, leave the
  config unchanged and still create or update the repo-local preference
- reject combining `--agents` and `--no-agents-warning`
- place generated `AGENTS.md` at the nearest Git repo root when the config is
  inside a repo; otherwise place it next to the selected config
- preserve existing `AGENTS.md` content by appending or refreshing only the
  marked Kitout section
- keep only deterministic directory resources active by default, including the
  parent directory for real Codex skill copies
- keep package, repo, copy, symlink, security, system, SSH key, and login-shell
  examples commented until the user customizes them

### `kitout context`

Prints agent-friendly context for the selected config without checking live
resource state or applying changes.

```sh
kitout context
kitout context --config ./kitout.yaml
kitout context --config ./kitout.yaml --json
```

Behavior:

- load and validate the selected config file
- print the selected config path and config directory
- summarize safe read-only commands an agent may use for evidence gathering
- summarize commands that require explicit user approval
- list declared managed resource IDs in config execution order
- include guidance for dotfiles repos, shell commands, dry-run, and secrets
- never call resource `Status` or `Apply`

Example human output:

```txt
Kitout agent context
Config: /Users/example/code/setup/kitout.yaml
Config directory: /Users/example/code/setup
Schema: version 1

Safe read-only commands:
- kitout context --config /Users/example/code/setup/kitout.yaml
- kitout status --config /Users/example/code/setup/kitout.yaml
- kitout apply --config /Users/example/code/setup/kitout.yaml --dry-run
- kitout upgrade --config /Users/example/code/setup/kitout.yaml --dry-run

Requires explicit user approval:
- kitout apply --config /Users/example/code/setup/kitout.yaml
- kitout upgrade --config /Users/example/code/setup/kitout.yaml
- configured shell resources

Managed resources:
- directory:/Users/example/code
- symlink:/Users/example/.zshrc
- shell:Enable Corepack

Agent guidance:
- Edit files in the setup repo or declared source paths, not managed targets in $HOME.
- Run status and dry-run commands before recommending a real apply or upgrade.
```

### `kitout status`

Checks configured resources.

```sh
kitout status
kitout status --json
kitout status --verbose
kitout status --config ./kitout.yaml
```

Current behavior:

- load and validate the selected config file
- print a startup line naming the selected config before status checks begin
- by default, show one progress line before batched Homebrew tap, package, and
  cask list fetches
- with `--verbose`, show a progress line before each resource status check
- build resources from config in stable execution order
- check resource status through the engine planner
- batch Homebrew tap, installed, and outdated checks across Homebrew resources during planning
- render resource details and summary counts
- render non-blocking advisories such as Homebrew formula or cask updates
- return `0` when all resources are satisfied or skipped
- return `1` when changes are needed
- return `2` for config validation, parse, or unknown-field errors
- return `3` for config read failures or failed/unknown resources
- when `--json` is passed, print machine-readable plan output

Example human output:

```txt
Kitout is checking your Mac setup...
Config: /Users/example/.config/kitout/kitout.yaml

> Fetching Homebrew tap list...
> Fetching Homebrew package list...
> Fetching Homebrew cask list...

Results:
✓ satisfied brew_tap: vwall/kitout        satisfied
✓ satisfied brew: git                    satisfied
✓ satisfied brew: go                     satisfied
            i brew: go: formula update available for go
                fix: Run `brew upgrade go` when you want to update it.
! missing   cask: ghostty                missing
✓ satisfied directory: /Users/example/code satisfied
- skip      repo: /Users/example/code/app skipped by config
× fail      shell: setup                 failed: command failed

Summary: 4 satisfied, 1 missing, 1 failed, 1 skipped
2 resources need attention
1 advisory
```

Exit behavior:

```txt
0 all resources satisfied
1 status found resources needing attention
2 validation error
3 runtime error
```

### `kitout apply`

Applies needed changes.

```sh
kitout apply
kitout apply --dry-run
kitout apply --yes
kitout apply --verbose
```

Behavior:

- load and validate config
- print a startup line naming the selected config before status checks begin
- show a progress line before each non-Homebrew resource status check
- show one Homebrew tap, formula, and cask inspection line for batched status checks
- run status checks
- build plan
- stop later apply actions when a blocking prerequisite, currently FileVault,
  cannot be satisfied
- leave failed or unknown resources unapplied, but continue with other planned
  missing or changed resources when they are not blocked by a prerequisite
- require confirmation before implemented risky actions, currently security
  changes, system prerequisite installers, SSH key generation, shell commands,
  login-shell changes, copy replacements, and symlink replacements, unless
  `--yes` is passed
- apply missing or changed resources in stable order
- show a progress line before each resource apply starts, so long-running commands such as Homebrew installs do not look stuck
- by default, capture subprocess output and include a compact stderr/stdout
  summary when a subprocess fails
- with `--verbose`, render each subprocess command and stream its stdout and stderr while it runs
- render summary
- return `0` when apply completed successfully
- return `2` for validation, parse, unknown-field, or flag errors
- return `3` for config read failures or interrupted planning/execution
- return `4` for failed/unknown resources or partial apply failures

### `kitout apply --dry-run`

Shows intended changes without making changes.

Example output:

```txt
[dry-run] Kitout is running in dry-run mode. No changes will be made.
Config: /Users/example/.config/kitout/kitout.yaml

> Inspecting Homebrew taps...
> Inspecting Homebrew casks...
> Checking copy: /Users/example/.codex/skills/nuxt-practices...
> Checking symlink: /Users/example/.zshrc...

[dry-run] Previewing planned changes:
dry-run Would add Homebrew tap vwall/kitout
dry-run Would install cask ghostty
dry-run Would copy to /Users/example/.codex/skills/nuxt-practices
dry-run Would replace symlink /Users/example/.zshrc

[dry-run] No changes made because --dry-run was used.
No shell commands will run without explicit approval.
```

### `kitout upgrade`

Upgrades outdated managed Homebrew formulae and casks. This command is separate
from `apply`: `apply` installs missing formulae and casks, while `upgrade`
refreshes installed managed formulae and casks only when Homebrew reports an
available update.

```sh
kitout upgrade
kitout upgrade --dry-run
kitout upgrade brew:git
kitout upgrade cask:ghostty
kitout upgrade --only brew
kitout upgrade --only cask
kitout upgrade --json
```

Behavior:

- load and validate config
- inspect only managed `brew.packages` and `brew.casks`
- show one Homebrew formula and cask inspection line for batched status checks
- treat installed outdated formulae and casks as planned upgrade actions
- skip missing formulae and casks with guidance to run `kitout apply` first
- leave current formulae and casks unchanged
- run `brew upgrade <name>` for outdated managed formulae
- run `brew upgrade --cask <name>` for outdated managed casks
- when resource IDs are passed, inspect and upgrade only those configured
  managed resources
- reject unknown, unsupported, or `--only`-excluded resource IDs before running
  Homebrew commands
- with `--only brew`, inspect and upgrade only managed formulae
- with `--only cask`, inspect and upgrade only managed casks
- with `--dry-run`, show planned upgrades without running `brew upgrade`
- with `--verbose`, render each subprocess command and stream its stdout and
  stderr while it runs
- return `0` when upgrade completed successfully or no upgrades are needed
- return `2` for validation, parse, unknown-field, or flag errors
- return `3` for config read failures, dry-run planning failures, or interrupted
  planning/execution
- return `4` for resource upgrade failures

Example dry-run output:

```txt
[dry-run] Kitout is checking managed Homebrew upgrades. No changes will be made.
Config: /Users/example/.config/kitout/kitout.yaml

> Inspecting Homebrew packages...
> Inspecting Homebrew casks...

[dry-run] Previewing managed upgrades:
dry-run Would upgrade formula go
dry-run Would upgrade cask ghostty
- Skipping cask: rectangle: cask is missing; run `kitout apply` first

[dry-run] No upgrades made because --dry-run was used.
```

### `kitout doctor`

Checks local prerequisites and common problems.

Homebrew freshness is checked by inspecting the local Homebrew repository's last
commit timestamp. A stale metadata warning tells the user to consider
`brew update`, but Kitout does not run `brew update` or upgrade packages
automatically.

```sh
kitout doctor
```

Checks:

- supported OS
- CPU architecture
- Xcode Command Line Tools
- Homebrew installation
- Homebrew install path for the current CPU architecture
- Homebrew metadata freshness
- Git installation
- shell environment
- config file validity
- write permissions for configured filesystem targets
- repo-local `AGENTS.md` presence when the selected config is inside a Git repo
  and `.kitout/agent-guidance.yaml` has not disabled the warning

If the command is interrupted, doctor stops dispatching checks and reports the
execution error separately instead of treating canceled commands as missing
prerequisites.

Current output:

```txt
Kitout is checking local prerequisites...
Config: /Users/me/.config/kitout/kitout.yaml

Doctor:
ok:   macOS                      running on macOS
ok:   CPU architecture           running on Apple Silicon
ok:   Xcode Command Line Tools   /Library/Developer/CommandLineTools
ok:   Homebrew                   Homebrew 4.0.0
ok:   Homebrew path              Homebrew prefix is /opt/homebrew
ok:   Homebrew freshness         Homebrew metadata looks current; last updated today
ok:   Git                        git version 2.45.0
ok:   Shell environment          SHELL and PATH look usable
ok:   Config                     config is valid
ok:   Path permissions           no configured filesystem write targets

10 total, 10 ok, 0 warnings, 0 failed
```

### `kitout explain <resource-id>`

Explains one configured resource by ID.

```sh
kitout explain --config ./kitout.yaml 'symlink:/Users/example/.zshrc'
kitout explain --config ./kitout.yaml --json 'shell:Enable Corepack'
```

Behavior:

- load and validate the selected config file
- find the requested resource ID
- run status for only that resource
- report state, planned action, details, and apply safety
- identify whether apply would require explicit user approval
- return `2` when the resource ID is not configured
- return `3` when the requested resource fails or cannot be inspected

Example human output:

```txt
Resource: symlink:/Users/example/.zshrc
Type: symlink
Config: /Users/example/code/setup/kitout.yaml

Current state:
  state: changed
  action: apply
  message: symlink points elsewhere

Details:
  replace: false
  source: /Users/example/code/setup/home/.zshrc
  target: /Users/example/.zshrc

Apply safety:
  would_apply: true
  requires_approval: true (symlink replacement may remove an existing target)
```

### `kitout version`

Prints version metadata.

```sh
kitout version
```

Current output:

```txt
kitout dev
commit unknown
built unknown
```

Release and local build artifacts should set `version`, `commit`, and `built`
metadata at build time.

## Output modes

### Human output

Default output should be concise and readable.

### JSON output

JSON output should be stable enough for tests and automation.

Status, apply, upgrade, doctor, context, and explain all use the same
envelope: `command`, `ok`, optional `config`, one command-specific payload, and
optional `error`. `upgrade --dry-run` uses the `plan` payload. A real
`upgrade --json` run uses the `upgrade` payload with the same summary and item
shape as `apply`.

Example status output:

```json
{
  "command": "status",
  "ok": true,
  "config": {
    "path": "/Users/example/.config/kitout/kitout.yaml",
    "valid": true
  },
  "plan": {
    "summary": {
      "total": 1,
      "satisfied": 0,
      "missing": 1,
      "changed": 0,
      "failed": 0,
      "skipped": 0,
      "unknown": 0,
      "to_apply": 1
    },
    "items": [
      {
        "resource_id": "directory:/Users/example/code",
        "type": "directory",
        "state": "missing",
        "action": "apply",
        "message": "directory is missing"
      }
    ]
  }
}
```

Example context payload:

```json
{
  "command": "context",
  "ok": true,
  "config": {
    "path": "/Users/example/code/setup/kitout.yaml",
    "valid": true
  },
  "context": {
    "schema_version": 1,
    "config_dir": "/Users/example/code/setup",
    "safe_commands": [
      {
        "command": "kitout status --config /Users/example/code/setup/kitout.yaml --json",
        "reason": "Return current resource state in stable JSON."
      }
    ],
    "requires_approval": [
      {
        "command": "kitout apply --config /Users/example/code/setup/kitout.yaml",
        "reason": "Applies changes to the user's machine."
      }
    ],
    "resources": [
      {
        "resource_id": "symlink:/Users/example/.zshrc",
        "type": "symlink",
        "label": "symlink: /Users/example/.zshrc",
        "details": {
          "source": "/Users/example/code/setup/home/.zshrc",
          "target": "/Users/example/.zshrc",
          "replace": "false"
        }
      }
    ],
    "guidance": [
      "Edit files in the setup repo or declared source paths, not managed targets in $HOME."
    ]
  }
}
```

Example explain payload:

```json
{
  "command": "explain",
  "ok": true,
  "config": {
    "path": "/Users/example/code/setup/kitout.yaml",
    "valid": true
  },
  "explain": {
    "resource": {
      "resource_id": "shell:Enable Corepack",
      "type": "shell",
      "label": "shell: Enable Corepack",
      "details": {
        "name": "Enable Corepack",
        "command": "corepack enable",
        "when": "missing-command:pnpm"
      }
    },
    "status": {
      "resource_id": "shell:Enable Corepack",
      "type": "shell",
      "state": "missing",
      "action": "apply",
      "message": "command should run",
      "details": {
        "name": "Enable Corepack",
        "command": "corepack enable",
        "when": "missing-command:pnpm"
      }
    },
    "safety": {
      "would_apply": true,
      "requires_approval": true,
      "reason": "shell resources run explicit configured commands during apply"
    }
  }
}
```

## Confirmation rules

Interactive confirmation is required for implemented risky changes unless
`--yes` is passed.

Risky changes currently include:

- replacing copy targets
- replacing symlink targets
- changing security settings
- starting system prerequisite installers
- generating SSH keys
- changing the current user's login shell
- running shell commands

Safe changes may apply without confirmation:

- installing a missing package
- creating a missing directory
- creating a missing copy target
- cloning a missing repo

Dry-run only renders the plan, so it does not ask for confirmation.

## Help text principle

Every command should have useful examples.

Avoid help text that only repeats the command name.
