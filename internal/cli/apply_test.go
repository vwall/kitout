package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyDryRunShowsPlanWithoutCreatingDirectory(t *testing.T) {
	missingDir := filepath.Join(t.TempDir(), "code")
	configPath := writeCLIConfigFile(t, `version: 1

directories:
  - `+missingDir+`
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"apply", "--config", configPath, "--dry-run"}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}
	if !strings.Contains(stdout.String(), "apply directory:"+missingDir) {
		t.Fatalf("stdout = %q, want dry-run directory plan", stdout.String())
	}
	if !strings.Contains(stdout.String(), "No changes made because --dry-run was used.") {
		t.Fatalf("stdout = %q, want dry-run no changes message", stdout.String())
	}
	if _, err := os.Stat(missingDir); !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) error = %v, want directory to remain missing", missingDir, err)
	}
}

func TestApplyCreatesMissingDirectory(t *testing.T) {
	missingDir := filepath.Join(t.TempDir(), "code")
	configPath := writeCLIConfigFile(t, `version: 1

directories:
  - `+missingDir+`
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"apply", "--config", configPath}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}
	if !strings.Contains(stdout.String(), "created directory") {
		t.Fatalf("stdout = %q, want created directory message", stdout.String())
	}
	info, err := os.Stat(missingDir)
	if err != nil {
		t.Fatalf("Stat(%q) returned error: %v", missingDir, err)
	}
	if !info.IsDir() {
		t.Fatalf("Stat(%q).IsDir() = false, want true", missingDir)
	}
}

func TestApplyReportsValidationErrors(t *testing.T) {
	configPath := writeCLIConfigFile(t, `version: 1

repos:
  - path: ~/code/kitout
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"apply", "--config", configPath}, nil, &stdout, &stderr)
	if code != exitValidation {
		t.Fatalf("exit code = %d, want %d", code, exitValidation)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Invalid config: repos[0].url is required") {
		t.Fatalf("stderr = %q, want structured validation error", stderr.String())
	}
}

func TestApplyJSONDryRunReportsPlan(t *testing.T) {
	missingDir := filepath.Join(t.TempDir(), "code")
	configPath := writeCLIConfigFile(t, `version: 1

directories:
  - `+missingDir+`
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"apply", "--config", configPath, "--dry-run", "--json"}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}
	response := decodeStatusJSON(t, stdout.String())
	if response.Command != "apply" {
		t.Fatalf("command = %q, want apply", response.Command)
	}
	if response.Plan == nil || !response.Plan.DryRun {
		t.Fatalf("plan = %+v, want dry-run plan", response.Plan)
	}
	if response.Plan.Summary.ToApply != 1 {
		t.Fatalf("ToApply = %d, want 1", response.Plan.Summary.ToApply)
	}
	if _, err := os.Stat(missingDir); !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) error = %v, want directory to remain missing", missingDir, err)
	}
}
