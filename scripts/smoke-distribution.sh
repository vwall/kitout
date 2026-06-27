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
login_shell_fixture_path=""
login_shell_current_shell=""
login_shell_smoke_enabled=false

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

parse_user_shell() {
  shell_output=$1
  while IFS= read -r line; do
    case "$line" in
      UserShell:*)
        value="${line#UserShell:}"
        value="${value#"${value%%[![:space:]]*}"}"
        value="${value%"${value##*[![:space:]]}"}"
        printf "%s\n" "$value"
        return 0
        ;;
    esac
  done <<EOF
$shell_output
EOF
  return 1
}

detect_login_shell_smoke_fixture() {
  if [ ! -x /usr/bin/id ] || [ ! -x /usr/bin/dscl ]; then
    echo "Skipping login-shell smoke: required system commands are unavailable."
    return 1
  fi

  if ! host_user="$(/usr/bin/id -un 2>/dev/null)"; then
    echo "Skipping login-shell smoke: could not determine current user."
    return 1
  fi
  if [ -z "$host_user" ]; then
    echo "Skipping login-shell smoke: current user name is empty."
    return 1
  fi

  if ! dscl_output="$(/usr/bin/dscl . -read "/Users/$host_user" UserShell 2>/dev/null)"; then
    echo "Skipping login-shell smoke: host cannot report UserShell for $host_user."
    return 1
  fi

  current_shell="$(parse_user_shell "$dscl_output")"
  if [ -z "$current_shell" ]; then
    echo "Skipping login-shell smoke: host UserShell output did not include a shell path."
    return 1
  fi

  for candidate in /bin/bash /bin/zsh; do
    if [ "$candidate" != "$current_shell" ] &&
      [ -x "$candidate" ] &&
      grep -Fxq "$candidate" /etc/shells; then
      login_shell_fixture_path="$candidate"
      login_shell_current_shell="$current_shell"
      return 0
    fi
  done

  echo "Skipping login-shell smoke: no safe standard shell differs from $current_shell."
  return 1
}

echo "Using temporary HOME: $tmp_home"
echo "Using temporary config: $config_path"
echo "Using temporary setup repo: $setup_dir"
echo "Using temporary local config: $local_config_path"
echo "Using temporary copy config: $copy_config_path"
echo "Using temporary login shell config: $login_shell_config_path"

expect_success init --config "$config_path"
expect_success init --config "$local_config_path"

if detect_login_shell_smoke_fixture; then
  login_shell_smoke_enabled=true
  echo "Using login shell smoke target: $login_shell_fixture_path (current: $login_shell_current_shell)"
fi

mkdir -p "$copy_source_dir/settings"
printf "font = \"Berkeley Mono\"\ntheme = \"system\"\n" > "$copy_source_dir/settings/editor.toml"
printf "Kitout smoke fixture\n" > "$copy_source_dir/README.txt"

cat > "$copy_config_path" <<EOF
version: 1

copies:
  - source: fixtures/editor-profile
    target: "~/Library/Application Support/Kitout Smoke/editor-profile"
EOF

if [ "$login_shell_smoke_enabled" = true ]; then
  cat > "$login_shell_config_path" <<EOF
version: 1

login_shell:
  path: "$login_shell_fixture_path"
  add_to_etc_shells: false
EOF
fi

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

if [ "$login_shell_smoke_enabled" = true ]; then
  expect_setup_exit_output_contains 1 "login shell differs" status --config "$login_shell_config_path"
  expect_setup_exit_output_contains 0 "Would set login shell to $login_shell_fixture_path" apply --config "$login_shell_config_path" --dry-run
fi

echo
echo "Kitout distribution smoke test passed."
