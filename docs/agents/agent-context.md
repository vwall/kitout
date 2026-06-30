# Agent Context

Kitout can give coding agents enough local evidence to answer setup questions
without guessing or changing the user's Mac.

Kitout does not embed AI behavior. The pattern is:

- Kitout owns desired state in `kitout.yaml`.
- Agents read Kitout context, status, doctor, dry-run, and explain output.
- Humans approve mutation before `kitout apply` or `kitout upgrade`.

## Repo Guidance

To give coding agents a durable, repo-local note about Kitout, generate
`AGENTS.md` from the setup repo:

```sh
kitout init --agents
```

When `--config` is omitted, `init` uses `./kitout.yaml` in the current
directory. With `--agents`, the generated section lists the safe Kitout
workflow, warns against secrets and unrequested shell commands, and points
agents to `kitout --help` instead of duplicating every command option. If
`AGENTS.md` already exists, Kitout preserves the existing content and appends or
refreshes only its marked Kitout section.

`kitout doctor --config ./kitout.yaml` reports a warning when the selected
config is inside a Git repo and `AGENTS.md` is missing. This is advisory only;
Kitout can still run without agent guidance.

If the repo intentionally does not want an `AGENTS.md` file, disable that
advisory without creating a placeholder:

```sh
kitout init --no-agents-warning
```

Kitout records the preference in `.kitout/agent-guidance.yaml` at the repo root.

## Safe Agent Loop

From a dotfiles or setup repo, select the repo config explicitly:

```sh
kitout context --config ./kitout.yaml
kitout doctor --config ./kitout.yaml --json
kitout status --config ./kitout.yaml --json
kitout apply --config ./kitout.yaml --dry-run --json
kitout upgrade --config ./kitout.yaml --dry-run --json
```

Use `explain` when the user asks about one resource:

```sh
kitout explain --config ./kitout.yaml 'symlink:/Users/example/.zshrc'
```

These commands are read-only or dry-run. They can inspect local state, but they
do not apply resources.

## What Agents Should Do

- Edit source files in the setup repo, not managed targets in `$HOME`.
- Run `kitout status` or `kitout apply --dry-run` before recommending apply.
- Run `kitout upgrade --dry-run` before recommending managed Homebrew upgrades.
- Ask before running `kitout apply` or `kitout upgrade`.
- Explain shell resources before any apply run.
- Never put secrets in Kitout config or managed dotfiles.

## Example Questions

If the user asks how to install `jq`, an agent can suggest:

```yaml
brew:
  packages:
    - jq
```

Then it should use:

```sh
kitout status --config ./kitout.yaml
kitout apply --config ./kitout.yaml --dry-run
```

If the user asks how to add a shell alias and Kitout manages `.zshrc` through a
symlink, the agent should edit the configured source path, not `~/.zshrc`.

## Output Surfaces

`kitout context` reports:

- selected config path
- safe read-only commands
- commands requiring explicit user approval
- declared resource IDs and details
- agent guidance

`kitout explain <resource-id>` reports:

- the requested resource ID and type
- current state and planned action
- resource details
- whether apply would require confirmation
- related commands to gather more evidence
