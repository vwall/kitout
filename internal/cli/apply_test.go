package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vwall/kitout/internal/engine"
	"github.com/vwall/kitout/internal/platform"
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

	observer.BeforeStatus(fakeCLIResource{id: "brew_tap:vwall/kitout", typ: "brew_tap"})
	observer.BeforeStatus(fakeCLIResource{id: "brew_tap:homebrew/cask-fonts", typ: "brew_tap"})
	observer.BeforeStatus(fakeCLIResource{id: "brew:asdf", typ: "brew"})
	observer.BeforeStatus(fakeCLIResource{id: "brew:git", typ: "brew"})
	observer.BeforeStatus(fakeCLIResource{id: "cask:ghostty", typ: "cask"})
	observer.BeforeStatus(fakeCLIResource{id: "cask:visual-studio-code", typ: "cask"})
	observer.BeforeStatus(fakeCLIResource{id: "/tmp/code", typ: "directory"})

	output := stdout.String()
	if count := strings.Count(output, "> Inspecting Homebrew taps..."); count != 1 {
		t.Fatalf("stdout = %q, want one Homebrew tap inspection line, got %d", output, count)
	}
	if count := strings.Count(output, "> Inspecting Homebrew packages..."); count != 1 {
		t.Fatalf("stdout = %q, want one Homebrew package inspection line, got %d", output, count)
	}
	if count := strings.Count(output, "> Inspecting Homebrew casks..."); count != 1 {
		t.Fatalf("stdout = %q, want one Homebrew cask inspection line, got %d", output, count)
	}
	if strings.Contains(output, "> Checking brew_tap: vwall/kitout...") || strings.Contains(output, "> Checking brew: asdf...") || strings.Contains(output, "> Checking cask: ghostty...") {
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

	observer.BeforeStatus(fakeCLIResource{id: "brew_tap:vwall/kitout", typ: "brew_tap"})
	observer.BeforeStatus(fakeCLIResource{id: "brew:asdf", typ: "brew"})
	observer.BeforeStatus(fakeCLIResource{id: "cask:ghostty", typ: "cask"})

	output := stdout.String()
	if !strings.Contains(output, "> Checking brew_tap: vwall/kitout...") {
		t.Fatalf("stdout = %q, want verbose brew tap status progress", output)
	}
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

func TestApplyJSONDryRunDoesNotRunShellCommand(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "marker")
	configPath := writeCLIConfigFile(t, `version: 1

shell:
  - name: Create marker
    command: touch `+marker+`
    when: always
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
	if response.Plan == nil || !response.Plan.DryRun || len(response.Plan.Items) != 1 {
		t.Fatalf("plan = %+v, want one dry-run shell item", response.Plan)
	}
	item := response.Plan.Items[0]
	if item.ResourceID != "shell:Create marker" || item.Action != "apply" {
		t.Fatalf("item = %+v, want shell apply plan item", item)
	}
	if item.Details["command"] != "touch "+marker {
		t.Fatalf("details = %+v, want shell command detail", item.Details)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) error = %v, want shell command marker to remain missing", marker, err)
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

func TestRiskyApplyItemsIncludesSecuritySystemAndSSHKeys(t *testing.T) {
	plan := engine.Plan{
		Items: []engine.PlanItem{
			{ResourceID: "security:firewall", Type: "security", State: engine.StateChanged, Action: engine.ActionApply},
			{ResourceID: "system:rosetta", Type: "system", State: engine.StateMissing, Action: engine.ActionApply},
			{ResourceID: "ssh_key:/Users/example/.ssh/id_ed25519", Type: "ssh_key", State: engine.StateMissing, Action: engine.ActionApply},
		},
	}

	items := riskyApplyItems(plan)

	if len(items) != 3 {
		t.Fatalf("len(items) = %d, want 3: %#v", len(items), items)
	}
	if items[0].Type != "security" {
		t.Fatalf("risky item[0] type = %q, want security", items[0].Type)
	}
	if items[1].Type != "system" {
		t.Fatalf("risky item[1] type = %q, want system", items[1].Type)
	}
	if items[2].Type != "ssh_key" {
		t.Fatalf("risky item[2] type = %q, want ssh_key", items[2].Type)
	}
}

func TestApplyRequiresConfirmationForSystemPrerequisite(t *testing.T) {
	planRunner := &fakeApplyRunner{responses: []fakeApplyResponse{
		{err: applyCommandError("xcode-select", []string{"-p"}, 2)},
	}}
	withCLIExecRunners(t, planRunner)
	configPath := writeCLIConfigFile(t, `version: 1

system:
  xcode_command_line_tools:
    required: true
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
	if !strings.Contains(stderr.String(), "system system:xcode_command_line_tools (xcode-select --install)") {
		t.Fatalf("stderr = %q, want system installer command detail", stderr.String())
	}
	expectApplyRunnerCalls(t, planRunner.calls, []applyCommandCall{
		{name: "xcode-select", args: []string{"-p"}},
	})
}

func TestApplyYesSkipsSystemPrerequisiteConfirmationAndRunsInstaller(t *testing.T) {
	planRunner := &fakeApplyRunner{responses: []fakeApplyResponse{
		{err: applyCommandError("xcode-select", []string{"-p"}, 2)},
	}}
	applyRunner := &fakeApplyRunner{responses: []fakeApplyResponse{
		{err: applyCommandError("xcode-select", []string{"-p"}, 2)},
		{result: applyCommandResult("xcode-select", []string{"--install"}, 0)},
	}}
	withCLIExecRunners(t, planRunner, applyRunner)
	configPath := writeCLIConfigFile(t, `version: 1

system:
  xcode_command_line_tools:
    required: true
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
	if !strings.Contains(stdout.String(), "started Command Line Tools installer") {
		t.Fatalf("stdout = %q, want installer apply result", stdout.String())
	}
	expectApplyRunnerCalls(t, planRunner.calls, []applyCommandCall{
		{name: "xcode-select", args: []string{"-p"}},
	})
	expectApplyRunnerCalls(t, applyRunner.calls, []applyCommandCall{
		{name: "xcode-select", args: []string{"-p"}},
		{name: "xcode-select", args: []string{"--install"}},
	})
}

func TestApplyDryRunShowsSystemPrerequisiteWithoutInstaller(t *testing.T) {
	planRunner := &fakeApplyRunner{responses: []fakeApplyResponse{
		{err: applyCommandError("xcode-select", []string{"-p"}, 2)},
	}}
	withCLIExecRunners(t, planRunner)
	configPath := writeCLIConfigFile(t, `version: 1

system:
  xcode_command_line_tools:
    required: true
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
	if !strings.Contains(stdout.String(), "dry-run Would start Command Line Tools installer") {
		t.Fatalf("stdout = %q, want system installer dry-run plan", stdout.String())
	}
	expectApplyRunnerCalls(t, planRunner.calls, []applyCommandCall{
		{name: "xcode-select", args: []string{"-p"}},
	})
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

func TestApplyShellCommandUsesUserPathForExplicitCommand(t *testing.T) {
	dir := t.TempDir()
	markerPath := filepath.Join(dir, "created")
	helperPath := filepath.Join(dir, "kitout-user-path-tool")
	if err := os.WriteFile(helperPath, []byte("#!/bin/sh\nprintf ok > \"$KITOUT_MARKER\"\n"), 0o755); err != nil {
		t.Fatalf("write helper: %v", err)
	}
	t.Setenv("KITOUT_MARKER", markerPath)
	t.Setenv("PATH", dir)
	configPath := writeCLIConfigFile(t, `version: 1

shell:
  - name: Run user path helper
    command: kitout-user-path-tool
    when: always
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"apply", "--config", configPath, "--yes"}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}
	if contents, err := os.ReadFile(markerPath); err != nil || string(contents) != "ok" {
		t.Fatalf("ReadFile(%q) = (%q, %v), want helper marker", markerPath, string(contents), err)
	}
}

func TestApplyShellMissingCommandConditionUsesUserPath(t *testing.T) {
	dir := t.TempDir()
	markerPath := filepath.Join(dir, "should-not-run")
	helperPath := filepath.Join(dir, "kitout-installed-tool")
	if err := os.WriteFile(helperPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write helper: %v", err)
	}
	t.Setenv("PATH", dir)
	configPath := writeCLIConfigFile(t, `version: 1

shell:
  - name: Do not rerun installed tool setup
    command: touch `+markerPath+`
    when: "missing-command: kitout-installed-tool"
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"apply", "--config", configPath, "--yes"}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}
	if _, err := os.Stat(markerPath); !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) error = %v, want command not to run", markerPath, err)
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

func TestApplyDryRunDoesNotResolveShellFromAmbientPathDuringStatus(t *testing.T) {
	dir := t.TempDir()
	markerPath := filepath.Join(dir, "path-runner-executed")
	fakeShellPath := filepath.Join(dir, "sh")
	if err := os.WriteFile(fakeShellPath, []byte("#!/bin/sh\nprintf compromised > \"$KITOUT_MARKER\"\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("write fake shell: %v", err)
	}
	t.Setenv("KITOUT_MARKER", markerPath)
	t.Setenv("PATH", dir)
	configPath := writeCLIConfigFile(t, `version: 1

shell:
  - name: Missing command check
    command: touch `+filepath.Join(dir, "created")+`
    when: "missing-command: kitout-definitely-missing-command"
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"apply", "--config", configPath, "--dry-run"}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}
	if _, err := os.Stat(markerPath); !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) error = %v, want ambient PATH shell not to run", markerPath, err)
	}
}

func TestApplyDryRunRejectsASDFToolVersionsSymlinkAncestor(t *testing.T) {
	dir := t.TempDir()
	linkedAncestor := filepath.Join(dir, "linked")
	outside := filepath.Join(dir, "outside")
	outsideTarget := filepath.Join(outside, ".tool-versions")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.Symlink(outside, linkedAncestor); err != nil {
		t.Fatalf("Symlink returned error: %v", err)
	}
	configPath := writeCLIConfigFile(t, `version: 1

asdf:
  tool_versions:
    - path: `+filepath.Join(linkedAncestor, ".tool-versions")+`
      tools:
        ruby: 3.3.6
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"apply", "--config", configPath, "--dry-run"}, nil, &stdout, &stderr)
	if code != exitRuntimeError {
		t.Fatalf("exit code = %d, want %d; stdout: %s; stderr: %s", code, exitRuntimeError, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), ".tool-versions ancestor") {
		t.Fatalf("stdout = %q, want .tool-versions ancestor failure", stdout.String())
	}
	if _, err := os.Lstat(outsideTarget); !os.IsNotExist(err) {
		t.Fatalf("Lstat(%q) error = %v, want outside target to remain missing", outsideTarget, err)
	}
}

func TestApplyDryRunRejectsCopySymlinkAncestor(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	linkedAncestor := filepath.Join(dir, "linked")
	outside := filepath.Join(dir, "outside")
	outsideTarget := filepath.Join(outside, "target")
	if err := os.WriteFile(source, []byte("contents\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.Symlink(outside, linkedAncestor); err != nil {
		t.Fatalf("Symlink returned error: %v", err)
	}
	configPath := writeCLIConfigFile(t, `version: 1

copies:
  - source: `+source+`
    target: `+filepath.Join(linkedAncestor, "target")+`
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"apply", "--config", configPath, "--dry-run"}, nil, &stdout, &stderr)
	if code != exitRuntimeError {
		t.Fatalf("exit code = %d, want %d; stdout: %s; stderr: %s", code, exitRuntimeError, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "copy target ancestor") {
		t.Fatalf("stdout = %q, want copy target ancestor failure", stdout.String())
	}
	if _, err := os.Lstat(outsideTarget); !os.IsNotExist(err) {
		t.Fatalf("Lstat(%q) error = %v, want outside target to remain missing", outsideTarget, err)
	}
}

func TestApplyRejectsDirectoryCopySymlinkAncestor(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	linkedAncestor := filepath.Join(dir, "linked")
	outside := filepath.Join(dir, "outside")
	outsideTarget := filepath.Join(outside, "profile", "settings.toml")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatalf("MkdirAll source returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "settings.toml"), []byte("theme = \"system\"\n"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatalf("MkdirAll outside returned error: %v", err)
	}
	if err := os.Symlink(outside, linkedAncestor); err != nil {
		t.Fatalf("Symlink returned error: %v", err)
	}
	configPath := writeCLIConfigFile(t, `version: 1

copies:
  - source: `+source+`
    target: `+filepath.Join(linkedAncestor, "profile")+`
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"apply", "--config", configPath}, nil, &stdout, &stderr)
	if code != exitApplyFailure {
		t.Fatalf("exit code = %d, want %d; stdout: %s; stderr: %s", code, exitApplyFailure, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "copy target ancestor") {
		t.Fatalf("stdout = %q, want copy target ancestor failure", stdout.String())
	}
	if _, err := os.Lstat(outsideTarget); !os.IsNotExist(err) {
		t.Fatalf("Lstat(%q) error = %v, want outside target to remain missing", outsideTarget, err)
	}
}

func TestApplyRejectsASDFToolVersionsSymlinkAncestor(t *testing.T) {
	dir := t.TempDir()
	linkedAncestor := filepath.Join(dir, "linked")
	outside := filepath.Join(dir, "outside")
	outsideTarget := filepath.Join(outside, ".tool-versions")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.Symlink(outside, linkedAncestor); err != nil {
		t.Fatalf("Symlink returned error: %v", err)
	}
	configPath := writeCLIConfigFile(t, `version: 1

asdf:
  tool_versions:
    - path: `+filepath.Join(linkedAncestor, ".tool-versions")+`
      tools:
        ruby: 3.3.6
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"apply", "--config", configPath}, nil, &stdout, &stderr)
	if code != exitApplyFailure {
		t.Fatalf("exit code = %d, want %d; stdout: %s; stderr: %s", code, exitApplyFailure, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), ".tool-versions ancestor") {
		t.Fatalf("stdout = %q, want .tool-versions ancestor failure", stdout.String())
	}
	if _, err := os.Lstat(outsideTarget); !os.IsNotExist(err) {
		t.Fatalf("Lstat(%q) error = %v, want outside target to remain missing", outsideTarget, err)
	}
}

func TestApplyDryRunRejectsSSHPublicKeySymlinkAncestor(t *testing.T) {
	dir := t.TempDir()
	linkedAncestor := filepath.Join(dir, "linked")
	outside := filepath.Join(dir, "outside")
	outsidePrivateKey := filepath.Join(outside, "id_ed25519")
	outsidePublicKey := outsidePrivateKey + ".pub"
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(outsidePrivateKey, []byte("private"), 0o600); err != nil {
		t.Fatalf("write private key: %v", err)
	}
	if err := os.Symlink(outside, linkedAncestor); err != nil {
		t.Fatalf("Symlink returned error: %v", err)
	}
	configPath := writeCLIConfigFile(t, `version: 1

ssh:
  keys:
    - path: `+filepath.Join(linkedAncestor, "id_ed25519")+`
      type: ed25519
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"apply", "--config", configPath, "--dry-run"}, nil, &stdout, &stderr)
	if code != exitRuntimeError {
		t.Fatalf("exit code = %d, want %d; stdout: %s; stderr: %s", code, exitRuntimeError, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "SSH public key ancestor") {
		t.Fatalf("stdout = %q, want SSH public key ancestor failure", stdout.String())
	}
	if _, err := os.Lstat(outsidePublicKey); !os.IsNotExist(err) {
		t.Fatalf("Lstat(%q) error = %v, want outside public key to remain missing", outsidePublicKey, err)
	}
}

func TestApplyRejectsSSHPublicKeySymlinkAncestor(t *testing.T) {
	dir := t.TempDir()
	linkedAncestor := filepath.Join(dir, "linked")
	outside := filepath.Join(dir, "outside")
	outsidePrivateKey := filepath.Join(outside, "id_ed25519")
	outsidePublicKey := outsidePrivateKey + ".pub"
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(outsidePrivateKey, []byte("private"), 0o600); err != nil {
		t.Fatalf("write private key: %v", err)
	}
	if err := os.Symlink(outside, linkedAncestor); err != nil {
		t.Fatalf("Symlink returned error: %v", err)
	}
	configPath := writeCLIConfigFile(t, `version: 1

ssh:
  keys:
    - path: `+filepath.Join(linkedAncestor, "id_ed25519")+`
      type: ed25519
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"apply", "--config", configPath, "--yes"}, nil, &stdout, &stderr)
	if code != exitApplyFailure {
		t.Fatalf("exit code = %d, want %d; stdout: %s; stderr: %s", code, exitApplyFailure, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "SSH public key ancestor") {
		t.Fatalf("stdout = %q, want SSH public key ancestor failure", stdout.String())
	}
	if _, err := os.Lstat(outsidePublicKey); !os.IsNotExist(err) {
		t.Fatalf("Lstat(%q) error = %v, want outside public key to remain missing", outsidePublicKey, err)
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

type applyCommandCall struct {
	name string
	args []string
}

type fakeApplyResponse struct {
	result platform.CommandResult
	err    error
}

type fakeApplyRunner struct {
	calls     []applyCommandCall
	responses []fakeApplyResponse
}

func (runner *fakeApplyRunner) Run(ctx context.Context, name string, args ...string) (platform.CommandResult, error) {
	runner.calls = append(runner.calls, applyCommandCall{
		name: name,
		args: append([]string(nil), args...),
	})

	if len(runner.responses) == 0 {
		return applyCommandResult(name, args, 0), nil
	}

	response := runner.responses[0]
	runner.responses = runner.responses[1:]
	if response.result.Name == "" {
		response.result = applyCommandResult(name, args, 0)
	}
	return response.result, response.err
}

func withCLIExecRunners(t *testing.T, runners ...platform.Runner) {
	t.Helper()

	original := cliExecRunnerFactory
	next := 0
	cliExecRunnerFactory = func() platform.Runner {
		if next >= len(runners) {
			t.Fatalf("newCLIExecRunner called %d times, want %d", next+1, len(runners))
		}
		runner := runners[next]
		next++
		return runner
	}
	t.Cleanup(func() {
		cliExecRunnerFactory = original
		if next != len(runners) {
			t.Fatalf("newCLIExecRunner called %d times, want %d", next, len(runners))
		}
	})
}

func expectApplyRunnerCalls(t *testing.T, got, want []applyCommandCall) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("calls = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i].name != want[i].name || strings.Join(got[i].args, "\x00") != strings.Join(want[i].args, "\x00") {
			t.Fatalf("calls = %#v, want %#v", got, want)
		}
	}
}

func applyCommandResult(name string, args []string, exitCode int) platform.CommandResult {
	return platform.CommandResult{
		Name:     name,
		Args:     append([]string(nil), args...),
		ExitCode: exitCode,
	}
}

func applyCommandError(name string, args []string, exitCode int) platform.CommandError {
	return platform.CommandError{
		Result: applyCommandResult(name, args, exitCode),
		Err:    errors.New("command failed"),
	}
}
