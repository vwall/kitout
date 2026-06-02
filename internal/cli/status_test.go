package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStatusLoadsValidConfig(t *testing.T) {
	configPath := writeCLIConfigFile(t, `version: 1

directories:
  - ~/code
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"status", "--config", configPath}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Config valid: "+configPath) {
		t.Fatalf("stdout = %q, want valid config path", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Status checks are not implemented yet.") {
		t.Fatalf("stdout = %q, want status stub message", stdout.String())
	}
}

func TestStatusReportsValidationErrors(t *testing.T) {
	configPath := writeCLIConfigFile(t, `version: 1

symlinks:
  - source: ~/dotfiles/zshrc
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"status", "--config", configPath}, nil, &stdout, &stderr)
	if code != exitValidation {
		t.Fatalf("exit code = %d, want %d", code, exitValidation)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Invalid config: symlinks[0].target is required") {
		t.Fatalf("stderr = %q, want structured validation error", stderr.String())
	}
}

func TestStatusReportsUnknownFieldsAsValidationErrors(t *testing.T) {
	configPath := writeCLIConfigFile(t, `version: 1
unknown: true
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"status", "--config", configPath}, nil, &stdout, &stderr)
	if code != exitValidation {
		t.Fatalf("exit code = %d, want %d", code, exitValidation)
	}
	if !strings.Contains(stderr.String(), "Invalid config: parse config "+configPath) {
		t.Fatalf("stderr = %q, want invalid config parse guidance", stderr.String())
	}
	if !strings.Contains(stderr.String(), "unknown") {
		t.Fatalf("stderr = %q, want unknown field name", stderr.String())
	}
}

func TestStatusReportsMissingConfigAsRuntimeError(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "missing.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"status", "--config", configPath}, nil, &stdout, &stderr)
	if code != exitRuntimeError {
		t.Fatalf("exit code = %d, want %d", code, exitRuntimeError)
	}
	if !strings.Contains(stderr.String(), "Failed to load config: read config "+configPath) {
		t.Fatalf("stderr = %q, want missing config guidance", stderr.String())
	}
}

func TestStatusAcceptsGlobalConfigFlagBeforeCommand(t *testing.T) {
	configPath := writeCLIConfigFile(t, "version: 1\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"--config", configPath, "status"}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}
}

func writeCLIConfigFile(t *testing.T, contents string) string {
	t.Helper()

	configPath := filepath.Join(t.TempDir(), "kitout.yaml")
	if err := os.WriteFile(configPath, []byte(contents), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	return configPath
}
