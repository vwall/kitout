# SSH Key Resource

## Purpose

The SSH key resource ensures a named SSH keypair exists without requiring a
custom shell block.

## Config

```yaml
ssh:
  keys:
    - path: ~/.ssh/id_ed25519
      type: ed25519
      comment: user@example.com
```

## Fields

```txt
path       required private key path
type       required key type, currently ed25519
comment    optional public key comment
```

## Status

Status reports:

- satisfied when both private and public key files exist, match each other, and
  match the configured key type
- missing when both the private key and `.pub` file are missing
- changed when the private key exists but the `.pub` file is missing
- failed when either key path points at a directory, cannot be inspected, or the
  public key does not match the private key
- failed when the private key is missing but `.pub` exists, because generating a
  new keypair would overwrite that public key

When both key files exist, status runs read-only `ssh-keygen -y` to confirm the
public key belongs to the private key.

## Apply

Apply generates a missing keypair with:

```sh
ssh-keygen -t ed25519 -C COMMENT -f PATH -N ''
```

When only the public key is missing, apply runs `ssh-keygen -y` to recreate it
from the private key.

Kitout never overwrites an existing private key or an existing `.pub` file when
the private key is missing. SSH key resources require confirmation during
`kitout apply` unless `--yes` is passed.

## Safety

The first implementation uses an empty passphrase so `kitout apply` can remain
non-interactive. Do not store passphrases or private key material in Kitout
config.

## Shared expectations

Every resource must support:

- status check
- apply
- dry-run plan
- readable result messages
- unit tests

Status must never change the system.

Apply must be idempotent.
