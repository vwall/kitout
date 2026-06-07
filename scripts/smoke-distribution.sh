#!/usr/bin/env bash
set -u
set -o pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
KITOUT_BIN="${KITOUT_BIN:-"$ROOT_DIR/bin/kitout"}"

if [ "$(uname -s)" != "Darwin" ]; then
  echo "kitout distribution smoke test is macOS-local; run it on macOS." >&2
  exit 1
fi

if [ ! -x "$KITOUT_BIN" ]; then
  echo "Kitout binary is not executable: $KITOUT_BIN" >&2
  echo "Run 'make build' first, or set KITOUT_BIN to a built kitout binary." >&2
  exit 1
fi

tmp_parent="${TMPDIR:-/tmp}"
tmp_root="$(mktemp -d "${tmp_parent%/}/kitout-smoke.XXXXXX")" || {
  echo "Could not create a temporary smoke-test directory." >&2
  exit 1
}
tmp_home="$tmp_root/home"
config_path="$tmp_home/.config/kitout/kitout.yaml"
setup_dir="$tmp_root/setup"
local_config_path="$setup_dir/kitout.yaml"

cleanup() {
  rm -rf "$tmp_root"
}
trap cleanup EXIT

mkdir -p "$tmp_home" "$setup_dir"
export SHELL="${SHELL:-/bin/zsh}"

run_kitout() {
  printf "\n==> kitout %s\n" "$*"
  HOME="$tmp_home" "$KITOUT_BIN" "$@" --no-color
}

run_kitout_in_setup() {
  printf "\n==> cd %s && kitout %s\n" "$setup_dir" "$*"
  (cd "$setup_dir" && HOME="$tmp_home" "$KITOUT_BIN" "$@" --no-color)
}

expect_success() {
  run_kitout "$@"
  code=$?
  if [ "$code" -ne 0 ]; then
    echo "Expected success, got exit code $code." >&2
    exit "$code"
  fi
}

expect_setup_exit_output_contains() {
  expected=$1
  fragment=$2
  shift 2

  output="$(run_kitout_in_setup "$@" 2>&1)"
  code=$?
  printf "%s\n" "$output"
  if [ "$code" -ne "$expected" ]; then
    echo "Expected exit code $expected, got $code." >&2
    exit 1
  fi
  case "$output" in
    *"$fragment"*) ;;
    *)
      echo "Expected output to contain: $fragment" >&2
      exit 1
      ;;
  esac
}

echo "Using temporary HOME: $tmp_home"
echo "Using temporary config: $config_path"
echo "Using temporary setup repo: $setup_dir"
echo "Using temporary local config: $local_config_path"

expect_success init --config "$config_path"
expect_success init --config "$local_config_path"

expect_setup_exit_output_contains 0 "Config: $local_config_path" doctor --config "$local_config_path"
expect_setup_exit_output_contains 0 "Config: $config_path" doctor --config "$config_path"

# The starter config intentionally includes ~/code. In the temporary HOME that
# directory is missing, so status should report planned work with exit code 1.
expect_setup_exit_output_contains 1 "Config: $local_config_path" status --config "$local_config_path"

expect_setup_exit_output_contains 0 "Config: $local_config_path" apply --config "$local_config_path" --dry-run
expect_setup_exit_output_contains 0 "Config: $config_path" apply --config "$config_path" --dry-run

echo
echo "Kitout distribution smoke test passed."
