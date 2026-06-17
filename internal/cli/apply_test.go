package cli

import (
	"bytes"
	"context"
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
	if !strings.HasPrefix(stdout.String(), "[dry-run] Kitout is running in dry-run mode. No changes will be made.\nConfig: "+configPath+"\n\n") {
		t.Fatalf("stdout = %q, want dry-run startup message", stdout.String())
	}
	if !strings.Contains(stdout.String(), "> Checking directory: "+missingDir+"...") {
		t.Fatalf("stdout = %q, want dry-run status check progress", stdout.String())
	}
	if !strings.Contains(stdout.String(), "[dry-run] Previewing planned changes:") {
		t.Fatalf("stdout = %q, want dry-run preview heading", stdout.String())
	}
	if !strings.Contains(stdout.String(), "dry-run Would create directory "+missingDir) {
		t.Fatalf("stdout = %q, want dry-run row marker", stdout.String())
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

func TestApplyDryRunColorCanBeForced(t *testing.T) {
	missingDir := filepath.Join(t.TempDir(), "code")
	configPath := writeCLIConfigFile(t, `version: 1

directories:
  - `+missingDir+`
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"apply", "--config", configPath, "--dry-run", "--color"}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}
	for _, fragment := range []string{
		ansiBlue + "[dry-run]" + ansiReset,
		ansiBlue + "dry-run" + ansiReset,
		ansiYellow + "Would create directory " + missingDir + ansiReset,
	} {
		if !strings.Contains(stdout.String(), fragment) {
			t.Fatalf("stdout = %q, want color fragment %q", stdout.String(), fragment)
		}
	}
}

func TestApplyDryRunUsesLocalConfigByDefaultWhenHomeConfigIsMissing(t *testing.T) {
	dir := t.TempDir()
	home := t.TempDir()
	t.Chdir(dir)
	t.Setenv("HOME", home)

	localDir := filepath.Join(dir, "local-code")
	localPath := filepath.Join(dir, "kitout.yaml")
	if err := os.WriteFile(localPath, []byte("version: 1\n\ndirectories:\n  - "+localDir+"\n"), 0o644); err != nil {
		t.Fatalf("write local config: %v", err)
	}
	wantLocalPath, err := filepath.Abs("kitout.yaml")
	if err != nil {
		t.Fatalf("resolve absolute local path: %v", err)
	}

	explicitDir := filepath.Join(dir, "explicit-code")
	explicitPath := filepath.Join(dir, "explicit.yaml")
	if err := os.WriteFile(explicitPath, []byte("version: 1\n\ndirectories:\n  - "+explicitDir+"\n"), 0o644); err != nil {
		t.Fatalf("write explicit config: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"apply", "--dry-run"}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Config: "+wantLocalPath) {
		t.Fatalf("stdout = %q, want local config path %q", stdout.String(), wantLocalPath)
	}
	if !strings.Contains(stdout.String(), "Would create directory "+localDir) {
		t.Fatalf("stdout = %q, want local directory plan", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()

	code = Run([]string{"apply", "--config", explicitPath, "--dry-run"}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Config: "+explicitPath) {
		t.Fatalf("stdout = %q, want explicit config path %q", stdout.String(), explicitPath)
	}
	if !strings.Contains(stdout.String(), "Would create directory "+explicitDir) {
		t.Fatalf("stdout = %q, want explicit directory plan", stdout.String())
	}
	if strings.Contains(stdout.String(), localDir) {
		t.Fatalf("stdout = %q, want explicit config to override local config", stdout.String())
	}
}

func TestApplyRejectsImplicitConfigWhenLocalAndHomeBothExist(t *testing.T) {
	dir := t.TempDir()
	home := t.TempDir()
	t.Chdir(dir)
	t.Setenv("HOME", home)

	localPath := filepath.Join(dir, "kitout.yaml")
	if err := os.WriteFile(localPath, []byte("version: 1\n"), 0o644); err != nil {
		t.Fatalf("write local config: %v", err)
	}
	homePath := writeHomeCLIConfigFile(t, home, "version: 1\n")
	wantLocalPath, err := filepath.Abs("kitout.yaml")
	if err != nil {
		t.Fatalf("resolve absolute local path: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"apply", "--dry-run"}, nil, &stdout, &stderr)
	if code != exitRuntimeError {
		t.Fatalf("exit code = %d, want %d", code, exitRuntimeError)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "both local config "+wantLocalPath+" and home config "+homePath+" exist") {
		t.Fatalf("stderr = %q, want ambiguous config guidance", stderr.String())
	}
	if !strings.Contains(stderr.String(), "pass --config to choose one") {
		t.Fatalf("stderr = %q, want --config guidance", stderr.String())
	}
}

func TestApplyJSONRejectsImplicitConfigWhenLocalAndHomeBothExist(t *testing.T) {
	dir := t.TempDir()
	home := t.TempDir()
	t.Chdir(dir)
	t.Setenv("HOME", home)

	if err := os.WriteFile(filepath.Join(dir, "kitout.yaml"), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatalf("write local config: %v", err)
	}
	writeHomeCLIConfigFile(t, home, "version: 1\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"apply", "--dry-run", "--json"}, nil, &stdout, &stderr)
	if code != exitRuntimeError {
		t.Fatalf("exit code = %d, want %d", code, exitRuntimeError)
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	response := decodeStatusJSON(t, stdout.String())
	if response.OK {
		t.Fatalf("ok = true, want false")
	}
	if response.Error == nil || response.Error.Type != "runtime" {
		t.Fatalf("error = %+v, want runtime error", response.Error)
	}
	if !strings.Contains(response.Error.Message, "pass --config to choose one") {
		t.Fatalf("message = %q, want --config guidance", response.Error.Message)
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
	if !strings.Contains(stdout.String(), "> Checking directory: "+missingDir+"...") {
		t.Fatalf("stdout = %q, want apply status check progress", stdout.String())
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

func TestApplyContinuesWhenAnotherResourceCannotBePlanned(t *testing.T) {
	dir := t.TempDir()
	missingDir := filepath.Join(dir, "code")
	missingCopySource := filepath.Join(dir, "missing-source")
	copyTarget := filepath.Join(dir, "copied")
	configPath := writeCLIConfigFile(t, `version: 1

directories:
  - `+missingDir+`

copies:
  - source: `+missingCopySource+`
    target: `+copyTarget+`
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"apply", "--config", configPath}, nil, &stdout, &stderr)
	if code != exitApplyFailure {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitApplyFailure, stderr.String())
	}
	if !strings.Contains(stdout.String(), "created directory") {
		t.Fatalf("stdout = %q, want directory apply result", stdout.String())
	}
	if !strings.Contains(stdout.String(), "copy source is missing") {
		t.Fatalf("stdout = %q, want failed copy result", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Summary: 1 changed, 1 failed") {
		t.Fatalf("stdout = %q, want partial failure summary", stdout.String())
	}
	info, err := os.Stat(missingDir)
	if err != nil {
		t.Fatalf("Stat(%q) returned error: %v", missingDir, err)
	}
	if !info.IsDir() {
		t.Fatalf("Stat(%q).IsDir() = false, want true", missingDir)
	}
	if _, err := os.Stat(copyTarget); !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) error = %v, want copy target to remain missing", copyTarget, err)
	}
}

func TestApplyPlanObserverShowsBatchedHomebrewProgressByDefault(t *testing.T) {
	var stdout bytes.Buffer
	renderer := newHumanRenderer(&stdout, &bytes.Buffer{}, globalOptions{})
	observer := newApplyPlanObserver(renderer, false)

	observer.BeforeStatus(fakeCLIResource{id: "brew:asdf", typ: "brew"})
	observer.BeforeStatus(fakeCLIResource{id: "brew:git", typ: "brew"})
	observer.BeforeStatus(fakeCLIResource{id: "cask:ghostty", typ: "cask"})
	observer.BeforeStatus(fakeCLIResource{id: "cask:visual-studio-code", typ: "cask"})
	observer.BeforeStatus(fakeCLIResource{id: "/tmp/code", typ: "directory"})

	output := stdout.String()
	if count := strings.Count(output, "> Inspecting Homebrew packages..."); count != 1 {
		t.Fatalf("stdout = %q, want one Homebrew package inspection line, got %d", output, count)
	}
	if count := strings.Count(output, "> Inspecting Homebrew casks..."); count != 1 {
		t.Fatalf("stdout = %q, want one Homebrew cask inspection line, got %d", output, count)
	}
	if strings.Contains(output, "> Checking brew: asdf...") || strings.Contains(output, "> Checking cask: ghostty...") {
		t.Fatalf("stdout = %q, want per-Homebrew checking lines hidden by default", output)
	}
	if !strings.Contains(output, "> Checking directory: /tmp/code...") {
		t.Fatalf("stdout = %q, want non-Homebrew status progress", output)
	}
}

func TestApplyPlanObserverVerboseShowsPerResourceProgress(t *testing.T) {
	var stdout bytes.Buffer
	renderer := newHumanRenderer(&stdout, &bytes.Buffer{}, globalOptions{})
	observer := newApplyPlanObserver(renderer, true)

	observer.BeforeStatus(fakeCLIResource{id: "brew:asdf", typ: "brew"})
	observer.BeforeStatus(fakeCLIResource{id: "cask:ghostty", typ: "cask"})

	output := stdout.String()
	if !strings.Contains(output, "> Checking brew: asdf...") {
		t.Fatalf("stdout = %q, want verbose brew status progress", output)
	}
	if !strings.Contains(output, "> Checking cask: ghostty...") {
		t.Fatalf("stdout = %q, want verbose cask status progress", output)
	}
	if strings.Contains(output, "Inspecting Homebrew") {
		t.Fatalf("stdout = %q, want verbose output to keep per-resource progress", output)
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

type fakeCLIResource struct {
	id  string
	typ string
}

func (resource fakeCLIResource) ID() string {
	return resource.id
}

func (resource fakeCLIResource) Type() string {
	return resource.typ
}

func (resource fakeCLIResource) Status(context.Context) (engine.StatusResult, error) {
	return engine.StatusResult{}, nil
}

func (resource fakeCLIResource) Apply(context.Context) (engine.ApplyResult, error) {
	return engine.ApplyResult{}, nil
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

func TestRiskyApplyItemsIncludesLoginShell(t *testing.T) {
	plan := engine.Plan{
		Items: []engine.PlanItem{
			{ResourceID: "directory:/tmp/code", Type: "directory", Action: engine.ActionApply},
			{ResourceID: "login_shell:homebrew:fish", Type: "login_shell", Action: engine.ActionApply},
		},
	}

	items := riskyApplyItems(plan)

	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1: %#v", len(items), items)
	}
	if items[0].Type != "login_shell" {
		t.Fatalf("risky item type = %q, want login_shell", items[0].Type)
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

func TestApplyRequiresConfirmationForCopyReplacement(t *testing.T) {
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

copies:
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
	if !strings.Contains(stderr.String(), "copy copy:"+targetPath) {
		t.Fatalf("stderr = %q, want copy replacement detail", stderr.String())
	}
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", targetPath, err)
	}
	if string(content) != "existing\n" {
		t.Fatalf("target content = %q, want existing file preserved", string(content))
	}
}

func TestApplyYesSkipsCopyReplacementConfirmation(t *testing.T) {
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

copies:
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
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", targetPath, err)
	}
	if string(content) != "source\n" {
		t.Fatalf("target content = %q, want source", string(content))
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
