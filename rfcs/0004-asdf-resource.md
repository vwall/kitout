# RFC 0004: asdf Resource

## Status

Draft

## Context

Kitout should support developer runtimes that are commonly managed with
`asdf`, such as Ruby and Node.js.

`asdf` uses plugins to manage different tools. A useful Kitout integration
needs to represent three pieces of desired state:

- an installed asdf plugin
- installed tool versions for that plugin
- explicit `.tool-versions` entries

This should be a built-in Kitout resource, not a Kitout plugin system.
Shell commands can run `asdf` manually, but they cannot give Kitout clear
status, dry-run, or idempotent apply behavior for plugins and versions.

Official asdf documentation is the reference for command behavior:

- https://asdf-vm.com/manage/plugins.html
- https://asdf-vm.com/manage/versions.html

## Decision

Add a future built-in `asdf` resource.

The first config shape should be:

```yaml
version: 1

brew:
  packages:
    - asdf

asdf:
  plugins:
    - name: ruby
      url: https://github.com/asdf-vm/asdf-ruby.git
      versions:
        - 3.3.6

  tool_versions:
    - path: ~/.tool-versions
      tools:
        ruby: 3.3.6
```

The `asdf` resource does not install `asdf` itself. Users who want Kitout to
install `asdf` should list it in `brew.packages`. If `asdf` is still not
available when the asdf resource runs, the resource should fail with an
actionable prerequisite message.

Plugin URLs are required. This avoids depending on the asdf short-name plugin
repository and makes the config more reproducible.

Only exact versions are supported in the first implementation. Do not support
`latest` in v1 because it makes status and apply behavior mutable over time.

`.tool-versions` files are updated only when explicitly configured. Kitout
must preserve unrelated entries in those files.

The fixed execution order should become:

1. doctor prerequisites
2. Homebrew packages
3. asdf plugins and versions
4. casks
5. directories
6. repositories
7. symlinks
8. macOS defaults
9. shell commands

## Status behavior

The asdf resource should check:

- `asdf` command availability
- plugin installed with the expected URL
- configured versions installed
- configured `.tool-versions` files contain expected tool/version entries

Missing `asdf` should be reported as a failed prerequisite, not as a plugin
or version failure.

## Apply behavior

The asdf resource should:

- add missing plugins with `asdf plugin add <name> <git-url>`
- install missing exact versions with `asdf install <name> <version>`
- update only configured `.tool-versions` files
- leave unrelated `.tool-versions` entries unchanged

The first implementation should not:

- install `asdf` directly
- remove asdf plugins
- uninstall tool versions
- update plugins
- rewrite unrelated `.tool-versions` entries
- support `latest`

## Dry-run behavior

Dry-run should show planned plugin additions, version installs, and
`.tool-versions` file updates.

Dry-run must not:

- add plugins
- install versions
- write `.tool-versions` files

## Testing expectations

Future implementation tests should cover:

- missing asdf prerequisite
- plugin satisfied
- plugin missing
- plugin installed with the wrong URL
- version satisfied
- version missing
- explicit `.tool-versions` entry satisfied
- explicit `.tool-versions` entry missing
- explicit `.tool-versions` entry changed
- dry-run command construction
- dry-run file update rendering
- no mutation during dry-run

Unit tests must not call a real `asdf` binary. External commands should use
the command runner interface.

## Consequences

Benefits:

- asdf-managed runtimes become visible in `kitout status`
- dry-run can show exact plugin, version, and file changes
- Ruby and other runtime setup can be represented declaratively
- shell commands remain a fallback instead of the primary path

Costs:

- config schema grows beyond the initial MVP resource list
- asdf plugin behavior may vary by tool
- `.tool-versions` editing requires careful preservation of unrelated entries

## Alternatives considered

### Shell commands

Rejected as the primary model. Shell commands are useful as an escape hatch,
but they do not provide reliable status checks or structured dry-run output
for asdf plugins and versions.

### Kitout plugin system

Rejected. This feature manages asdf plugins as local machine state. It should
not introduce a plugin API for Kitout itself.

### Install asdf inside the asdf resource

Rejected for the first implementation. Homebrew package installation already
belongs to the Homebrew resource. Keeping asdf installation there makes the
asdf resource independent and easier to test.

### Optional plugin URLs

Rejected for the first implementation. Required URLs are more explicit and
avoid relying on the asdf short-name plugin repository.
