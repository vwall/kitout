# ASDF Resource

## Purpose

The asdf resources ensure developer runtime plugins, exact runtime versions, and
explicit `.tool-versions` entries are present.

The asdf resources do not install asdf itself. Install asdf with Homebrew first:

```yaml
brew:
  packages:
    - asdf
```

Because Kitout checks all resources before applying any of them, a first dry-run
on a fresh machine can report asdf plugin resources as failed until `asdf` is
available on `PATH`. Install or apply the prerequisite first, then rerun
`kitout status` or `kitout apply --dry-run`. If the full config contains failed
asdf plugin resources, `kitout apply` stops before changing anything; use a
small bootstrap config or temporarily leave dependent asdf sections out until
`asdf` is installed.

## Config

```yaml
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

Plugin URLs are required so Kitout does not depend on the asdf short-name plugin
repository. Versions must be exact; `latest` is rejected.

## Status check

Plugin status checks:

```sh
asdf --version
asdf plugin list --urls
asdf list <name>
```

Satisfied when the plugin URL matches the config and all configured versions
are installed.

Missing when the plugin or one of its configured versions is missing.

Changed when the plugin is installed from a different URL.

Failed when asdf is unavailable or an asdf command errors unexpectedly.

`.tool-versions` status checks read the configured file and compare only the
configured tool entries.

## Apply

Plugin apply may run:

```sh
asdf plugin add <name> <url>
asdf install <name> <version>
```

`.tool-versions` apply writes only configured tool entries. Unrelated entries
and comments are preserved.

The first implementation does not:

- install asdf directly
- remove plugins
- uninstall versions
- update plugins
- support `latest`

## Implementation status

Implemented as `resources.ASDFPluginResource` and
`resources.ASDFToolVersionsResource`. External commands use the shared command
runner interface so tests do not call a real asdf binary.
