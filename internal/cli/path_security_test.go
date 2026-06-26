package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStatusDoesNotResolveBrewFromAmbientPath(t *testing.T) {
	dir := t.TempDir()
	markerPath := filepath.Join(dir, "fake-brew-ran")
	writeFakePathTool(t, dir, "brew", `#!/bin/sh
printf compromised > "$KITOUT_MARKER"
printf kitout-security-test\\n
exit 0
`)
	t.Setenv("KITOUT_MARKER", markerPath)
	t.Setenv("PATH", dir)
	configPath := writeCLIConfigFile(t, `version: 1
brew:
  packages:
    - kitout-security-test
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	Run([]string{"status", "--config", configPath, "--no-color"}, nil, &stdout, &stderr)

	assertPathToolDidNotRun(t, markerPath, stdout.String(), stderr.String())
}

func TestApplyDryRunDoesNotResolveBrewFromAmbientPath(t *testing.T) {
	dir := t.TempDir()
	markerPath := filepath.Join(dir, "fake-brew-ran")
	writeFakePathTool(t, dir, "brew", `#!/bin/sh
printf compromised > "$KITOUT_MARKER"
printf kitout-security-test\\n
exit 0
`)
	t.Setenv("KITOUT_MARKER", markerPath)
	t.Setenv("PATH", dir)
	configPath := writeCLIConfigFile(t, `version: 1
brew:
  packages:
    - kitout-security-test
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	Run([]string{"apply", "--config", configPath, "--dry-run", "--no-color"}, nil, &stdout, &stderr)

	assertPathToolDidNotRun(t, markerPath, stdout.String(), stderr.String())
}

func TestDoctorDoesNotResolvePrerequisitesFromAmbientPath(t *testing.T) {
	dir := t.TempDir()
	markerPath := filepath.Join(dir, "fake-prerequisites-ran")
	script := `#!/bin/sh
printf '%s\n' "$0" >> "$KITOUT_MARKER"
name="${0##*/}"
case "$name" in
  xcode-select) printf /fake/xcode\\n ;;
  brew)
    if [ "${1:-}" = "--prefix" ]; then
      printf /opt/homebrew\\n
    else
      printf 'Homebrew 9.9.9\n'
    fi
    ;;
  git) printf 'git version 9.9.9\n' ;;
esac
exit 0
`
	writeFakePathTool(t, dir, "xcode-select", script)
	writeFakePathTool(t, dir, "brew", script)
	writeFakePathTool(t, dir, "git", script)
	t.Setenv("KITOUT_MARKER", markerPath)
	t.Setenv("PATH", dir)
	t.Setenv("SHELL", executablePath(t))
	configPath := writeCLIConfigFile(t, "version: 1\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	Run([]string{"doctor", "--config", configPath, "--no-color"}, nil, &stdout, &stderr)

	assertPathToolDidNotRun(t, markerPath, stdout.String(), stderr.String())
}

func writeFakePathTool(t *testing.T, dir, name, script string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake %s: %v", name, err)
	}
	return path
}

func assertPathToolDidNotRun(t *testing.T, markerPath, stdout, stderr string) {
	t.Helper()

	if _, err := os.Stat(markerPath); !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) error = %v, want ambient PATH tool not to run", markerPath, err)
	}
	if strings.Contains(stdout, "compromised") || strings.Contains(stderr, "compromised") {
		t.Fatalf("fake PATH tool output leaked into command output\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
}
