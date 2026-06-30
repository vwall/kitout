package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/vwall/kitout/internal/platform"
)

func TestUpgradeDryRunShowsManagedHomebrewUpgrades(t *testing.T) {
	runner := &fakeApplyRunner{responses: []fakeApplyResponse{
		{result: applyResultWithStdout("brew", []string{"list", "--formula", "--quiet"}, "git\ngo\n")},
		{result: applyResultWithStdout("brew", []string{"outdated", "--formula", "--quiet"}, "go\n")},
		{result: applyResultWithStdout("brew", []string{"list", "--cask", "--quiet"}, "ghostty\n")},
		{result: applyResultWithStdout("brew", []string{"outdated", "--cask", "--quiet"}, "ghostty\n")},
	}}
	withCLIExecRunners(t, runner)
	configPath := writeCLIConfigFile(t, `version: 1

brew:
  packages:
    - git
    - go
  casks:
    - ghostty
    - rectangle

directories:
  - /tmp/kitout-upgrade-ignored
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"upgrade", "--config", configPath, "--dry-run"}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}
	for _, fragment := range []string{
		"[dry-run] Kitout is checking managed Homebrew upgrades. No changes will be made.",
		"> Inspecting Homebrew packages...",
		"> Inspecting Homebrew casks...",
		"[dry-run] Previewing managed upgrades:",
		"Would upgrade formula go",
		"Would upgrade cask ghostty",
		"Skipping cask: rectangle: cask is missing; run `kitout apply` first",
		"No upgrades made because --dry-run was used.",
	} {
		if !strings.Contains(stdout.String(), fragment) {
			t.Fatalf("stdout = %q, want fragment %q", stdout.String(), fragment)
		}
	}
	if strings.Contains(stdout.String(), "kitout-upgrade-ignored") {
		t.Fatalf("stdout = %q, want non-Homebrew resources ignored", stdout.String())
	}
	expectApplyRunnerCalls(t, runner.calls, []applyCommandCall{
		{name: "brew", args: []string{"list", "--formula", "--quiet"}},
		{name: "brew", args: []string{"outdated", "--formula", "--quiet"}},
		{name: "brew", args: []string{"list", "--cask", "--quiet"}},
		{name: "brew", args: []string{"outdated", "--cask", "--quiet"}},
	})
}

func TestUpgradeAppliesManagedHomebrewUpgrades(t *testing.T) {
	planRunner := &fakeApplyRunner{responses: []fakeApplyResponse{
		{result: applyResultWithStdout("brew", []string{"list", "--formula", "--quiet"}, "git\n")},
		{result: applyResultWithStdout("brew", []string{"outdated", "--formula", "--quiet"}, "git\n")},
		{result: applyResultWithStdout("brew", []string{"list", "--cask", "--quiet"}, "ghostty\n")},
		{result: applyResultWithStdout("brew", []string{"outdated", "--cask", "--quiet"}, "ghostty\n")},
	}}
	upgradeRunner := &fakeApplyRunner{responses: []fakeApplyResponse{
		{result: applyCommandResult("brew", []string{"list", "--formula", "git"}, 0)},
		{result: applyResultWithStdout("brew", []string{"outdated", "--formula", "--quiet"}, "git\n")},
		{result: applyCommandResult("brew", []string{"upgrade", "git"}, 0)},
		{result: applyCommandResult("brew", []string{"list", "--cask", "ghostty"}, 0)},
		{result: applyResultWithStdout("brew", []string{"outdated", "--cask", "--quiet"}, "ghostty\n")},
		{result: applyCommandResult("brew", []string{"upgrade", "--cask", "ghostty"}, 0)},
	}}
	withCLIExecRunners(t, planRunner, upgradeRunner)
	configPath := writeCLIConfigFile(t, `version: 1

brew:
  packages:
    - git
  casks:
    - ghostty
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"upgrade", "--config", configPath}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}
	for _, fragment := range []string{
		"Kitout is planning managed Homebrew upgrades...",
		"Upgrading managed Homebrew resources:",
		"> Upgrading formula git...",
		"> Upgrading cask ghostty...",
		"brew: git",
		"upgraded formula",
		"cask: ghostty",
		"upgraded cask",
		"Summary: 2 changed",
	} {
		if !strings.Contains(stdout.String(), fragment) {
			t.Fatalf("stdout = %q, want fragment %q", stdout.String(), fragment)
		}
	}
	expectApplyRunnerCalls(t, planRunner.calls, []applyCommandCall{
		{name: "brew", args: []string{"list", "--formula", "--quiet"}},
		{name: "brew", args: []string{"outdated", "--formula", "--quiet"}},
		{name: "brew", args: []string{"list", "--cask", "--quiet"}},
		{name: "brew", args: []string{"outdated", "--cask", "--quiet"}},
	})
	expectApplyRunnerCalls(t, upgradeRunner.calls, []applyCommandCall{
		{name: "brew", args: []string{"list", "--formula", "git"}},
		{name: "brew", args: []string{"outdated", "--formula", "--quiet"}},
		{name: "brew", args: []string{"upgrade", "git"}},
		{name: "brew", args: []string{"list", "--cask", "ghostty"}},
		{name: "brew", args: []string{"outdated", "--cask", "--quiet"}},
		{name: "brew", args: []string{"upgrade", "--cask", "ghostty"}},
	})
}

func TestUpgradeOnlyBrewSkipsCaskChecks(t *testing.T) {
	runner := &fakeApplyRunner{responses: []fakeApplyResponse{
		{result: applyResultWithStdout("brew", []string{"list", "--formula", "--quiet"}, "git\n")},
		{result: applyResultWithStdout("brew", []string{"outdated", "--formula", "--quiet"}, "git\n")},
	}}
	withCLIExecRunners(t, runner)
	configPath := writeCLIConfigFile(t, `version: 1

brew:
  packages:
    - git
  casks:
    - ghostty
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"upgrade", "--config", configPath, "--dry-run", "--only", "brew"}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Would upgrade formula git") {
		t.Fatalf("stdout = %q, want formula upgrade", stdout.String())
	}
	if strings.Contains(stdout.String(), "cask") {
		t.Fatalf("stdout = %q, want cask resources omitted", stdout.String())
	}
	expectApplyRunnerCalls(t, runner.calls, []applyCommandCall{
		{name: "brew", args: []string{"list", "--formula", "--quiet"}},
		{name: "brew", args: []string{"outdated", "--formula", "--quiet"}},
	})
}

func TestUpgradeDryRunCanTargetManagedResourceID(t *testing.T) {
	runner := &fakeApplyRunner{responses: []fakeApplyResponse{
		{result: applyResultWithStdout("brew", []string{"list", "--formula", "--quiet"}, "git\ngo\n")},
		{result: applyResultWithStdout("brew", []string{"outdated", "--formula", "--quiet"}, "go\n")},
	}}
	withCLIExecRunners(t, runner)
	configPath := writeCLIConfigFile(t, `version: 1

brew:
  packages:
    - git
    - go
  casks:
    - ghostty
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"upgrade", "--config", configPath, "--dry-run", "brew:go"}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Would upgrade formula go") {
		t.Fatalf("stdout = %q, want targeted formula upgrade", stdout.String())
	}
	for _, fragment := range []string{"brew: git", "cask", "ghostty"} {
		if strings.Contains(stdout.String(), fragment) {
			t.Fatalf("stdout = %q, want %q omitted", stdout.String(), fragment)
		}
	}
	expectApplyRunnerCalls(t, runner.calls, []applyCommandCall{
		{name: "brew", args: []string{"list", "--formula", "--quiet"}},
		{name: "brew", args: []string{"outdated", "--formula", "--quiet"}},
	})
}

func TestUpgradeDryRunJSONReportsPlan(t *testing.T) {
	runner := &fakeApplyRunner{responses: []fakeApplyResponse{
		{result: applyResultWithStdout("brew", []string{"list", "--formula", "--quiet"}, "git\n")},
		{result: applyResultWithStdout("brew", []string{"outdated", "--formula", "--quiet"}, "git\n")},
	}}
	withCLIExecRunners(t, runner)
	configPath := writeCLIConfigFile(t, `version: 1

brew:
  packages:
    - git
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"upgrade", "--config", configPath, "--dry-run", "--json"}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}
	response := decodeStatusJSON(t, stdout.String())
	if response.Command != "upgrade" {
		t.Fatalf("command = %q, want upgrade", response.Command)
	}
	if response.Plan == nil || !response.Plan.DryRun {
		t.Fatalf("plan = %+v, want dry-run plan", response.Plan)
	}
	if response.Plan.Summary.ToApply != 1 || response.Plan.Summary.Changed != 1 {
		t.Fatalf("summary = %+v, want one changed upgrade", response.Plan.Summary)
	}
	if got := response.Plan.Items[0].Action; got != "apply" {
		t.Fatalf("action = %q, want apply", got)
	}
	if got := response.Plan.Items[0].State; got != "changed" {
		t.Fatalf("state = %q, want changed", got)
	}
}

func TestUpgradeJSONReportsUpgradeResults(t *testing.T) {
	planRunner := &fakeApplyRunner{responses: []fakeApplyResponse{
		{result: applyResultWithStdout("brew", []string{"list", "--formula", "--quiet"}, "git\n")},
		{result: applyResultWithStdout("brew", []string{"outdated", "--formula", "--quiet"}, "git\n")},
	}}
	upgradeRunner := &fakeApplyRunner{responses: []fakeApplyResponse{
		{result: applyCommandResult("brew", []string{"list", "--formula", "git"}, 0)},
		{result: applyResultWithStdout("brew", []string{"outdated", "--formula", "--quiet"}, "git\n")},
		{result: applyCommandResult("brew", []string{"upgrade", "git"}, 0)},
	}}
	withCLIExecRunners(t, planRunner, upgradeRunner)
	configPath := writeCLIConfigFile(t, `version: 1

brew:
  packages:
    - git
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"upgrade", "--config", configPath, "--json"}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}
	response := decodeStatusJSON(t, stdout.String())
	if response.Command != "upgrade" {
		t.Fatalf("command = %q, want upgrade", response.Command)
	}
	if response.Upgrade == nil {
		t.Fatalf("upgrade report = nil, want report")
	}
	if response.Upgrade.Summary.Changed != 1 {
		t.Fatalf("summary = %+v, want one changed", response.Upgrade.Summary)
	}
	if got := response.Upgrade.Items[0].Action; got != "upgrade" {
		t.Fatalf("action = %q, want upgrade", got)
	}
}

func TestUpgradeRejectsUnknownOnlyFilter(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"upgrade", "--only", "repos"}, nil, &stdout, &stderr)
	if code != exitValidation {
		t.Fatalf("exit code = %d, want %d", code, exitValidation)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), `kitout upgrade: --only must be brew or cask, got "repos"`) {
		t.Fatalf("stderr = %q, want --only validation", stderr.String())
	}
}

func TestUpgradeRejectsUnknownTarget(t *testing.T) {
	runner := &fakeApplyRunner{}
	withCLIExecRunners(t, runner)
	configPath := writeCLIConfigFile(t, `version: 1

brew:
  packages:
    - git
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"upgrade", "--config", configPath, "brew:go"}, nil, &stdout, &stderr)
	if code != exitValidation {
		t.Fatalf("exit code = %d, want %d", code, exitValidation)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), `kitout upgrade: unknown upgrade target "brew:go"`) {
		t.Fatalf("stderr = %q, want unknown target validation", stderr.String())
	}
	if len(runner.calls) != 0 {
		t.Fatalf("calls = %+v, want no Homebrew commands", runner.calls)
	}
}

func applyResultWithStdout(name string, args []string, stdout string) platform.CommandResult {
	result := applyCommandResult(name, args, 0)
	result.Stdout = stdout
	return result
}
