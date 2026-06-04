package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vwall/kitout/internal/engine"
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
	if !strings.HasPrefix(stdout.String(), "Kitout is planning changes only. No changes will be made.\nConfig: "+configPath+"\n\n") {
		t.Fatalf("stdout = %q, want dry-run startup message", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Would create directory "+missingDir) {
		t.Fatalf("stdout = %q, want dry-run directory plan", stdout.String())
	}
	if !strings.Contains(stdout.String(), "No changes made because --dry-run was used.") {
		t.Fatalf("stdout = %q, want dry-run no changes message", stdout.String())
	}
	if !strings.Contains(stdout.String(), "No shell commands will run without explicit approval.") {
		t.Fatalf("stdout = %q, want dry-run safety message", stdout.String())
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
	if strings.Contains(stderr.String(), "Risky apply actions require confirmation") {
		t.Fatalf("stderr = %q, want no confirmation prompt for missing directory", stderr.String())
	}
	if !strings.HasPrefix(stdout.String(), "Kitout is planning changes for your Mac setup...\nConfig: "+configPath+"\n\n") {
		t.Fatalf("stdout = %q, want apply startup message", stdout.String())
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

func TestApplyRequiresConfirmationForShellCommand(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "created")
	configPath := writeCLIConfigFile(t, `version: 1

shell:
  - name: Create marker
    command: touch `+outputPath+`
    when: always
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"apply", "--config", configPath}, strings.NewReader("no\n"), &stdout, &stderr)
	if code != exitValidation {
		t.Fatalf("exit code = %d, want %d", code, exitValidation)
	}
	if !strings.Contains(stderr.String(), "Risky apply actions require confirmation") {
		t.Fatalf("stderr = %q, want confirmation prompt", stderr.String())
	}
	if !strings.Contains(stderr.String(), "touch "+outputPath) {
		t.Fatalf("stderr = %q, want shell command detail", stderr.String())
	}
	if _, err := os.Stat(outputPath); !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) error = %v, want command not to run", outputPath, err)
	}
}

func TestApplyRunsShellCommandAfterConfirmation(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "created")
	configPath := writeCLIConfigFile(t, `version: 1

shell:
  - name: Create marker
    command: touch `+outputPath+`
    when: always
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"apply", "--config", configPath}, strings.NewReader("yes\n"), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}
	if !strings.Contains(stderr.String(), "Risky apply actions require confirmation") {
		t.Fatalf("stderr = %q, want confirmation prompt", stderr.String())
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("Stat(%q) error = %v, want command to run", outputPath, err)
	}
}

func TestApplyYesSkipsShellCommandConfirmation(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "created")
	configPath := writeCLIConfigFile(t, `version: 1

shell:
  - name: Create marker
    command: touch `+outputPath+`
    when: always
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"apply", "--config", configPath, "--yes"}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}
	if strings.Contains(stderr.String(), "Risky apply actions require confirmation") {
		t.Fatalf("stderr = %q, want no confirmation prompt", stderr.String())
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("Stat(%q) error = %v, want command to run", outputPath, err)
	}
}

func TestApplyKeepsShellCommandOutputConciseByDefault(t *testing.T) {
	configPath := writeCLIConfigFile(t, `version: 1

shell:
  - name: Print verbose output
    command: printf kitout-stdout; printf kitout-stderr >&2
    when: always
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"apply", "--config", configPath, "--yes"}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}
	if strings.Contains(stdout.String(), "kitout-stdout") {
		t.Fatalf("stdout = %q, want subprocess stdout hidden by default", stdout.String())
	}
	if strings.Contains(stderr.String(), "kitout-stderr") {
		t.Fatalf("stderr = %q, want subprocess stderr hidden by default", stderr.String())
	}
}

func TestApplyVerboseStreamsShellCommandOutput(t *testing.T) {
	configPath := writeCLIConfigFile(t, `version: 1

shell:
  - name: Print verbose output
    command: printf kitout-stdout; printf kitout-stderr >&2
    when: always
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"apply", "--config", configPath, "--yes", "--verbose"}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}
	if !strings.Contains(stdout.String(), "$ sh -c 'printf kitout-stdout; printf kitout-stderr >&2'") {
		t.Fatalf("stdout = %q, want rendered shell command", stdout.String())
	}
	if !strings.Contains(stdout.String(), "kitout-stdout") {
		t.Fatalf("stdout = %q, want streamed subprocess stdout", stdout.String())
	}
	if !strings.Contains(stderr.String(), "kitout-stderr") {
		t.Fatalf("stderr = %q, want streamed subprocess stderr", stderr.String())
	}
}

func TestApplyDryRunDoesNotPromptForShellCommand(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "created")
	configPath := writeCLIConfigFile(t, `version: 1

shell:
  - name: Create marker
    command: touch `+outputPath+`
    when: always
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"apply", "--config", configPath, "--dry-run"}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}
	if strings.Contains(stderr.String(), "Risky apply actions require confirmation") {
		t.Fatalf("stderr = %q, want no confirmation prompt during dry-run", stderr.String())
	}
	if _, err := os.Stat(outputPath); !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) error = %v, want command not to run", outputPath, err)
	}
}

func TestApplyRequiresConfirmationForSymlinkReplacement(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "source")
	targetPath := filepath.Join(dir, "target")
	if err := os.WriteFile(sourcePath, []byte("source\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := os.WriteFile(targetPath, []byte("existing\n"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	configPath := writeCLIConfigFile(t, `version: 1

symlinks:
  - source: `+sourcePath+`
    target: `+targetPath+`
    replace: true
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"apply", "--config", configPath}, strings.NewReader("no\n"), &stdout, &stderr)
	if code != exitValidation {
		t.Fatalf("exit code = %d, want %d", code, exitValidation)
	}
	if !strings.Contains(stderr.String(), "symlink symlink:"+targetPath) {
		t.Fatalf("stderr = %q, want symlink replacement detail", stderr.String())
	}
	info, err := os.Lstat(targetPath)
	if err != nil {
		t.Fatalf("Lstat(%q) error = %v", targetPath, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("target became a symlink after aborted apply")
	}
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", targetPath, err)
	}
	if string(content) != "existing\n" {
		t.Fatalf("target content = %q, want existing file preserved", string(content))
	}
}

func TestApplyYesSkipsSymlinkReplacementConfirmation(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "source")
	targetPath := filepath.Join(dir, "target")
	if err := os.WriteFile(sourcePath, []byte("source\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := os.WriteFile(targetPath, []byte("existing\n"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	configPath := writeCLIConfigFile(t, `version: 1

symlinks:
  - source: `+sourcePath+`
    target: `+targetPath+`
    replace: true
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"apply", "--config", configPath, "--yes"}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}
	if strings.Contains(stderr.String(), "Risky apply actions require confirmation") {
		t.Fatalf("stderr = %q, want no confirmation prompt", stderr.String())
	}
	info, err := os.Lstat(targetPath)
	if err != nil {
		t.Fatalf("Lstat(%q) error = %v", targetPath, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("target is not a symlink after apply")
	}
	linkTarget, err := os.Readlink(targetPath)
	if err != nil {
		t.Fatalf("Readlink(%q) error = %v", targetPath, err)
	}
	if linkTarget != sourcePath {
		t.Fatalf("Readlink(%q) = %q, want %q", targetPath, linkTarget, sourcePath)
	}
}

func TestRiskyApplyItemsExcludesMacOSDefaults(t *testing.T) {
	plan := engine.Plan{
		Items: []engine.PlanItem{
			{
				ResourceID: "macos_default:NSGlobalDomain/AppleShowAllExtensions",
				Type:       "macos_default",
				State:      engine.StateChanged,
				Action:     engine.ActionApply,
			},
		},
	}

	items := riskyApplyItems(plan)

	if len(items) != 0 {
		t.Fatalf("len(items) = %d, want 0", len(items))
	}
}
