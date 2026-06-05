# Design QA

final result: passed

Reference target: blended retro-modern Macintosh direction with Version 3's terminal-first layout and Version 2's warm carbon/manual palette.

Checks completed:

- Desktop preview loaded at `http://127.0.0.1:8000/`.
- Mobile preview checked at `390 x 844`.
- No horizontal overflow on desktop or mobile.
- Hero asset loads successfully.
- Mobile navigation opens and reports `aria-expanded`.
- Visible command examples use `kitout status`, `kitout apply --dry-run`, `kitout apply`, and docs links; unsupported planning subcommands do not appear.
- Visible examples use `/Users/example/...` paths and do not include local personal paths.
- HTML parsed with Python's standard `html.parser`.

Remaining P3 polish:

- The hero terminal sits partly below the first desktop viewport by design; it preserves the large editorial first impression while inviting scroll.
