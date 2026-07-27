## Summary

Describe the user-visible change and why it belongs in Kitout.

## Verification

- [ ] `make fmt-check`
- [ ] `make test`
- [ ] `make vet`
- [ ] `make release-check` when changing runtime, release, or macOS behavior

## Safety

- [ ] Status and dry-run remain read-only.
- [ ] Apply remains idempotent.
- [ ] Risky changes require explicit configuration and confirmation.
- [ ] No secrets or private machine details are included.

## Documentation

- [ ] CLI help, docs, and examples are updated when behavior or configuration changes.
