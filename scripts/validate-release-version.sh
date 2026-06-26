#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -ne 1 ]; then
  echo "usage: validate-release-version.sh <tag>" >&2
  exit 2
fi

tag="$1"
version="${tag#v}"

if [ "$tag" = "$version" ]; then
  echo "release tag must start with v" >&2
  exit 1
fi

if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "release tag must be vX.Y.Z with numeric components" >&2
  exit 1
fi

printf "%s\n" "$version"
