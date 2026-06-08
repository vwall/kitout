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
copy_config_path="$setup_dir/kitout.copy.yaml"
login_shell_config_path="$setup_dir/kitout.login-shell.yaml"
copy_source_dir="$setup_dir/fixtures/editor-profile"
copy_target_dir="$tmp_home/Library/Application Support/Kitout Smoke/editor-profile"
copy_target_label="~/Library/Application Support/Kitout Smoke/editor-profile"
login_shell_shim_dir="$tmp_root/login-shell-shims"

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

run_kitout_in_setup_with_login_shell_shims() {
  printf "\n==> cd %s && kitout %s\n" "$setup_dir" "$*"
  (cd "$setup_dir" && HOME="$tmp_home" PATH="$login_shell_shim_dir:$PATH" "$KITOUT_BIN" "$@" --no-color)
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

expect_login_shell_exit_output_contains() {
  expected=$1
  fragment=$2
  shift 2

  output="$(run_kitout_in_setup_with_login_shell_shims "$@" 2>&1)"
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
echo "Using temporary copy config: $copy_config_path"
echo "Using temporary login shell config: $login_shell_config_path"

expect_success init --config "$config_path"
expect_success init --config "$local_config_path"

login_shell_fixture_path="/bin/bash"
login_shell_reported_current="/bin/zsh"
if [ ! -x "$login_shell_fixture_path" ] || [ ! -x "$login_shell_reported_current" ]; then
  echo "Could not find safe standard login shell paths for smoke coverage." >&2
  exit 1
fi
if ! grep -Fxq "$login_shell_fixture_path" /etc/shells || ! grep -Fxq "$login_shell_reported_current" /etc/shells; then
  echo "Could not find safe standard login shells in /etc/shells for smoke coverage." >&2
  exit 1
fi

mkdir -p "$copy_source_dir/settings" "$login_shell_shim_dir"
printf "font = \"Berkeley Mono\"\ntheme = \"system\"\n" > "$copy_source_dir/settings/editor.toml"
printf "Kitout smoke fixture\n" > "$copy_source_dir/README.txt"

cat > "$login_shell_shim_dir/id" <<'EOF'
#!/usr/bin/env bash
if [ "${1:-}" = "-un" ]; then
  printf "kitout-smoke\n"
  exit 0
fi
exec /usr/bin/id "$@"
EOF

cat > "$login_shell_shim_dir/dscl" <<EOF
#!/usr/bin/env bash
if [ "\${1:-}" = "." ] && [ "\${2:-}" = "-read" ] && [ "\${3:-}" = "/Users/kitout-smoke" ] && [ "\${4:-}" = "UserShell" ]; then
  printf "UserShell: %s\n" "$login_shell_reported_current"
  exit 0
fi
printf "unexpected dscl invocation: %s\n" "\$*" >&2
exit 1
EOF
chmod +x "$login_shell_shim_dir/id" "$login_shell_shim_dir/dscl"

cat > "$copy_config_path" <<EOF
version: 1

copies:
  - source: fixtures/editor-profile
    target: "~/Library/Application Support/Kitout Smoke/editor-profile"
EOF

cat > "$login_shell_config_path" <<EOF
version: 1

login_shell:
  path: "$login_shell_fixture_path"
  add_to_etc_shells: false
EOF

expect_setup_exit_output_contains 0 "Config: $local_config_path" doctor --config "$local_config_path"
expect_setup_exit_output_contains 0 "Config: $config_path" doctor --config "$config_path"

# The starter config intentionally includes ~/code. In the temporary HOME that
# directory is missing, so status should report planned work with exit code 1.
expect_setup_exit_output_contains 1 "Config: $local_config_path" status --config "$local_config_path"

expect_setup_exit_output_contains 0 "Config: $local_config_path" apply --config "$local_config_path" --dry-run
expect_setup_exit_output_contains 0 "Config: $config_path" apply --config "$config_path" --dry-run

expect_setup_exit_output_contains 1 "copy: $copy_target_label" status --config "$copy_config_path"
expect_setup_exit_output_contains 0 "Would copy to $copy_target_label" apply --config "$copy_config_path" --dry-run
if [ -e "$copy_target_dir" ]; then
  echo "Dry-run unexpectedly created copy target: $copy_target_dir" >&2
  exit 1
fi
expect_setup_exit_output_contains 0 "copied source to target" apply --config "$copy_config_path" --yes
if ! diff -ru "$copy_source_dir" "$copy_target_dir"; then
  echo "Copied fixture does not match source fixture." >&2
  exit 1
fi
expect_setup_exit_output_contains 0 "Summary: 1 satisfied" status --config "$copy_config_path"

expect_login_shell_exit_output_contains 1 "login shell differs" status --config "$login_shell_config_path"
expect_login_shell_exit_output_contains 0 "Would set login shell to $login_shell_fixture_path" apply --config "$login_shell_config_path" --dry-run

echo
echo "Kitout distribution smoke test passed."
