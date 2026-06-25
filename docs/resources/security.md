# Security Resources

## Purpose

Security resources manage or require a small set of macOS security states that
are better represented as typed Kitout config than shell gates.

## Config

```yaml
security:
  filevault:
    required: true
  firewall:
    enabled: true
    stealth_mode: true
```

## Fields

```txt
filevault.required       required true when filevault is configured
firewall.enabled         required target firewall state
firewall.stealth_mode    optional target stealth mode state
```

`firewall.stealth_mode` requires `firewall.enabled: true`.

## FileVault

Status runs `fdesetup status` and reports satisfied only when FileVault is on.

Apply does not enable FileVault automatically. FileVault setup involves
user-specific recovery-key and account choices, so Kitout opens System Settings
and returns a manual-action failure. Enable FileVault, then rerun Kitout.

Because `filevault.required` is a prerequisite, an unmet or uninspectable
FileVault requirement blocks later `kitout apply` actions. Kitout should not
install packages, write files, or run shell commands until FileVault is enabled.

## Firewall

Status uses `/usr/libexec/ApplicationFirewall/socketfilterfw` to inspect the
global firewall and stealth-mode states.

Apply uses `sudo /usr/libexec/ApplicationFirewall/socketfilterfw` to update
states that differ from config. Security resources require confirmation during
`kitout apply` unless `--yes` is passed.

## Shared expectations

Every resource must support:

- status check
- apply
- dry-run plan
- readable result messages
- unit tests

Status must never change the system.

Apply must be idempotent.
