# System Resources

## Purpose

System resources manage macOS prerequisites that fresh developer machines often
need before package, repository, or runtime setup works reliably.

## Config

```yaml
system:
  xcode_command_line_tools:
    required: true
  rosetta:
    required: true
```

## Fields

```txt
xcode_command_line_tools.required    required true when configured
rosetta.required                     required true when configured
```

## Xcode Command Line Tools

Status runs `xcode-select -p`.

Apply runs `xcode-select --install` when the tools are missing. macOS may show a
GUI installer; rerun Kitout after that installer completes.

## Rosetta

Status runs `uname -m` first. On non-Apple-Silicon Macs, Rosetta is treated as
satisfied because it is not required. On Apple Silicon, status checks the
`com.apple.pkg.RosettaUpdateAuto` package receipt.

Apply runs:

```sh
softwareupdate --install-rosetta --agree-to-license
```

## Shared expectations

Every resource must support:

- status check
- apply
- dry-run plan
- readable result messages
- unit tests

Status must never change the system.

Apply must be idempotent.
