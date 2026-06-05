#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -ne 3 ]; then
  echo "usage: prepare-release-notes.sh <tag> <version> <output-file>" >&2
  exit 2
fi

tag="$1"
version="$2"
output_file="$3"

version_notes="docs/release/${version}.md"
tag_notes="docs/release/${tag}.md"

if [ -f "${version_notes}" ]; then
  cp "${version_notes}" "${output_file}"
  printf "%s\n" "${version_notes}"
  exit 0
fi

if [ -f "${tag_notes}" ]; then
  cp "${tag_notes}" "${output_file}"
  printf "%s\n" "${tag_notes}"
  exit 0
fi

cat > "${output_file}" <<EOF
# Kitout ${version}

No release notes file was found for ${tag}.

Release assets:

- kitout_${version}_darwin_arm64.tar.gz
- kitout_${version}_darwin_amd64.tar.gz
- kitout_${version}_checksums.txt
EOF

printf "generated fallback\n"
