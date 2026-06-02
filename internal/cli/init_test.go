package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCreatesDefaultConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"init"}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}

	configPath := filepath.Join(home, ".config", "kitout", "kitout.yaml")
	contents, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("expected config file to be created: %v", err)
	}

	if !strings.Contains(string(contents), "version: 1") {
		t.Fatalf("starter config missing version: %s", string(contents))
	}

	if !strings.Contains(stdout.String(), configPath) {
		t.Fatalf("stdout = %q, want created path %q", stdout.String(), configPath)
	}
}

func TestInitRefusesToOverwriteExistingConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "kitout.yaml")
	if err := os.WriteFile(configPath, []byte("version: 1\ncustom: true\n"), 0o644); err != nil {
		t.Fatalf("write existing config: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"init", "--config", configPath}, nil, &stdout, &stderr)
	if code != exitValidation {
		t.Fatalf("exit code = %d, want %d", code, exitValidation)
	}

	contents, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read existing config: %v", err)
	}

	if string(contents) != "version: 1\ncustom: true\n" {
		t.Fatalf("existing config was changed: %q", string(contents))
	}

	if !strings.Contains(stderr.String(), "Use --force to overwrite it.") {
		t.Fatalf("stderr = %q, want overwrite guidance", stderr.String())
	}
}

func TestInitForceOverwritesExistingConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "kitout.yaml")
	if err := os.WriteFile(configPath, []byte("version: 1\ncustom: true\n"), 0o644); err != nil {
		t.Fatalf("write existing config: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"init", "--config", configPath, "--force"}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}

	contents, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read overwritten config: %v", err)
	}

	if strings.Contains(string(contents), "custom: true") {
		t.Fatalf("existing config was not overwritten: %q", string(contents))
	}

	if !strings.Contains(string(contents), "brew:") {
		t.Fatalf("starter config missing brew section: %q", string(contents))
	}
}

func TestInitAcceptsGlobalConfigFlagBeforeCommand(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "kitout.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"--config", configPath, "init"}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}

	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("expected config file to be created at global --config path: %v", err)
	}
}
