# Name and Registry Review

Review date: 2026-06-05

This is a lightweight launch-readiness review, not legal advice or formal trademark clearance.

## Result

The `kitout` name remains acceptable for a public 0.1.0 developer-tool preview, with one caveat: before broader promotion, run a formal USPTO trademark search or legal review for software/developer-tool use.

## Checks Performed

- Homebrew: local `brew search --formula kitout` returned no matching formula, and `brew search --cask kitout` did not return a `kitout` cask.
- Homebrew public formula browser/API: web searches for an exact `kitout` formula did not identify an existing Homebrew core formula.
- PyPI/npm: public web searches did not identify a prominent existing package named exactly `kitout`, but the first release does not use PyPI or npm distribution.
- GitHub/repository naming: the active source repository is `github.com/vwall/kitout`.
- General naming: `kitout` is a plain-English phrase also used in clothing, workwear, and outfitting contexts; the `.dev` positioning and developer-machine scope reduce product-category ambiguity.

## Sources to Recheck Before Tagging

- Homebrew formula browser: https://formulae.brew.sh/
- Homebrew taps documentation: https://docs.brew.sh/Taps
- PyPI project URL: https://pypi.org/project/kitout/
- npm package URL: https://www.npmjs.com/package/kitout
- USPTO search portal: https://www.uspto.gov/trademarks/search

## Recommendation

Proceed with `Kitout` for `v0.1.0`, keep the product positioned as a macOS developer setup CLI, and do not claim trademark ownership beyond ordinary project naming until a formal clearance pass is complete.
