# Product

## Register

product

## Users

Kitout is for developers who want a Mac to reach a known-good setup without replaying manual notes, fragile shell scripts, or scattered dotfile steps.

Primary contexts:

- setting up a fresh Mac after a new machine, reset, or job change
- checking whether a daily-use machine has drifted from the desired setup
- adding a package, cask, runtime, repository, directory, symlink, macOS default, or approved shell command to a shared personal setup config
- debugging setup prerequisites before applying changes

Users are usually in a terminal, often before their full development environment is ready. They need concise status, safe previews, clear failures, and exact next actions.

## Product Purpose

Kitout is a Go CLI that reconciles a developer Mac against a declarative YAML setup file.

It exists to make machine setup repeatable, inspectable, and safe to rerun. Success means a user can run `kitout status`, understand what is satisfied or missing, preview changes with `kitout apply --dry-run`, apply only intended changes with `kitout apply`, and diagnose prerequisites with `kitout doctor`.

Kitout should sit between dotfile managers, one-off setup scripts, and heavyweight reproducibility systems: broader than symlink management, safer than imperative scripts, and smaller than a full device-management platform.

## Brand Personality

Kitout should feel practical, friendly, calm, safe, and confident.

The product voice should be direct and specific. It should explain what happened, what will happen, what failed, and how to fix common problems. It should avoid theatrical language, inflated claims, and cleverness that makes setup behavior harder to trust.

## Anti-references

Kitout should not look, read, or behave like:

- an AI-branded assistant or automation platform
- enterprise device-management software
- a magical setup tool that hides system changes
- a Bash DSL compatibility layer
- an arbitrary shell scripting framework
- a secret manager
- a plugin-first or extensibility-first product
- a destructive installer that overwrites user files by default

Avoid positioning language such as productivity platform, cloud control plane, endpoint management, autonomous, agentic, or magic.

## Design Principles

1. Show desired state plainly.
   Every status and apply view should make it obvious which resources are satisfied, missing, changed, failed, skipped, or planned.

2. Preview before changing.
   Dry-run is a core product promise. Users should be able to inspect intended changes without wondering whether checks mutated the machine.

3. Make safety visible.
   Risky operations should name the target path, command, or replacement before they run. File overwrites, shell commands, and profile modifications need explicit consent or configuration.

4. Keep the command surface boring.
   The CLI should favor predictable commands, stable flags, structured results, and readable summaries over novelty.

5. Let failures teach the next step.
   Doctor warnings, validation errors, and apply failures should tell users what is wrong and how to resolve it without burying the answer in verbose logs.

## Accessibility & Inclusion

Kitout should work well in ordinary terminals, headless environments, CI logs, and accessible terminal setups.

The CLI should preserve meaning without color, support `--no-color`, keep human output scannable with text markers and aligned labels, and provide JSON output for tooling. Error messages should avoid relying on symbols alone. Any future visual surface should meet WCAG 2.2 AA contrast expectations, respect reduced-motion preferences, and keep status semantics available to assistive technology.
