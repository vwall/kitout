# Homebrew Formula

This directory contains the draft formula for the external Homebrew tap.

The intended tap is:

```sh
brew tap vwall/kitout
brew install kitout
```

Homebrew maps `brew tap vwall/kitout` to `github.com/vwall/homebrew-kitout`.

## Release Update Steps

1. Tag and publish `v0.1.0` from the main repository.
2. Download or inspect `kitout_0.1.0_checksums.txt` from the GitHub release.
3. Copy `kitout.rb.template` to `github.com/vwall/homebrew-kitout` as `Formula/kitout.rb`.
4. Replace `ARM64_SHA256_FROM_RELEASE` and `AMD64_SHA256_FROM_RELEASE` with the matching checksums from the release assets.
5. Run:

```sh
brew audit --strict --online kitout
brew install ./Formula/kitout.rb
brew test kitout
```

The formula must use checksums from the exact tarballs uploaded by the GitHub release workflow.
