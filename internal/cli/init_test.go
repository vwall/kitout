package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vwall/kitout/internal/config"
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

func TestInitGeneratedConfigLoadsAndStatusParsesWithoutEdits(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	configPath := filepath.Join(home, ".config", "kitout", "kitout.yaml")

	var initStdout bytes.Buffer
	var initStderr bytes.Buffer

	code := Run([]string{"init"}, nil, &initStdout, &initStderr)
	if code != exitOK {
		t.Fatalf("init exit code = %d, want %d; stderr: %s", code, exitOK, initStderr.String())
	}

	loaded, err := config.LoadFile(configPath)
	if err != nil {
		t.Fatalf("generated config did not load and validate: %v", err)
	}
	if loaded.Path != configPath {
		t.Fatalf("loaded path = %q, want %q", loaded.Path, configPath)
	}
	if len(loaded.Config.Directories) != 3 {
		t.Fatalf("generated config directories = %#v, want exactly three safe active directories", loaded.Config.Directories)
	}
	if got, want := loaded.Config.Directories[0], filepath.Join(home, "code"); got != want {
		t.Fatalf("first generated directory = %q, want %q", got, want)
	}
	if got, want := loaded.Config.Directories[2], filepath.Join(home, ".codex", "skills"); got != want {
		t.Fatalf("third generated directory = %q, want %q", got, want)
	}
	if len(loaded.Config.Copies) != 0 {
		t.Fatalf("generated config active copies = %#v, want none", loaded.Config.Copies)
	}
	if len(loaded.Config.Symlinks) != 0 {
		t.Fatalf("generated config active symlinks = %#v, want none", loaded.Config.Symlinks)
	}
	if loaded.Config.LoginShell != nil {
		t.Fatalf("generated config active login_shell = %#v, want nil", loaded.Config.LoginShell)
	}
	if loaded.Config.Security.FileVault != nil || loaded.Config.Security.Firewall != nil {
		t.Fatalf("generated config active security = %#v, want none", loaded.Config.Security)
	}
	if loaded.Config.System.XcodeCommandLineTools != nil || loaded.Config.System.Rosetta != nil {
		t.Fatalf("generated config active system = %#v, want none", loaded.Config.System)
	}
	if len(loaded.Config.SSH.Keys) != 0 {
		t.Fatalf("generated config active SSH keys = %#v, want none", loaded.Config.SSH.Keys)
	}
	contents := string(mustReadFile(t, configPath))
	if !strings.Contains(contents, "# copies:") {
		t.Fatalf("generated config missing commented copies example")
	}
	if !strings.Contains(contents, "# security:") {
		t.Fatalf("generated config missing commented security example")
	}
	if !strings.Contains(contents, "# system:") {
		t.Fatalf("generated config missing commented system example")
	}
	if !strings.Contains(contents, "# ssh:") {
		t.Fatalf("generated config missing commented ssh example")
	}
	if !strings.Contains(contents, "# login_shell:") {
		t.Fatalf("generated config missing commented login_shell example")
	}

	var statusStdout bytes.Buffer
	var statusStderr bytes.Buffer

	code = Run([]string{"status", "--config", configPath}, nil, &statusStdout, &statusStderr)
	if code != exitChanges {
		t.Fatalf("status exit code = %d, want %d; stdout: %s; stderr: %s", code, exitChanges, statusStdout.String(), statusStderr.String())
	}
	if statusStderr.String() != "" {
		t.Fatalf("status stderr = %q, want empty", statusStderr.String())
	}
	if !strings.Contains(statusStdout.String(), "directory: ~/code") {
		t.Fatalf("status stdout = %q, want generated directory resource", statusStdout.String())
	}
}

func TestInitAgentsCreatesRepoGuidance(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatalf("create .git directory: %v", err)
	}
	configPath := filepath.Join(dir, "kitout.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"init", "--config", configPath, "--agents"}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}

	agentsPath := filepath.Join(dir, "AGENTS.md")
	contents := string(mustReadFile(t, agentsPath))
	for _, fragment := range []string{
		"# AGENTS.md",
		"## Kitout",
		"kitout status --config ./kitout.yaml",
		"kitout apply --config ./kitout.yaml --dry-run",
		"Use `kitout --help` and `kitout <command> --help`",
	} {
		if !strings.Contains(contents, fragment) {
			t.Fatalf("AGENTS.md = %q, want fragment %q", contents, fragment)
		}
	}
	if !strings.Contains(stdout.String(), "Created AGENTS.md: "+agentsPath) {
		t.Fatalf("stdout = %q, want created AGENTS path %q", stdout.String(), agentsPath)
	}
}

func TestInitAgentsUsesNestedConfigPathInRepoGuidance(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatalf("create .git directory: %v", err)
	}
	configPath := filepath.Join(dir, "configs", "kitout.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"init", "--config", configPath, "--agents"}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}

	agentsPath := filepath.Join(dir, "AGENTS.md")
	contents := string(mustReadFile(t, agentsPath))
	if !strings.Contains(contents, "kitout status --config ./configs/kitout.yaml") {
		t.Fatalf("AGENTS.md = %q, want nested config path", contents)
	}
	if strings.Contains(contents, "kitout status --config ./kitout.yaml") {
		t.Fatalf("AGENTS.md = %q, want no root config path for nested config", contents)
	}
}

func TestInitAgentsUpdatesExistingRepoGuidanceWithoutOverwritingConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatalf("create .git directory: %v", err)
	}
	configPath := filepath.Join(dir, "kitout.yaml")
	configContents := "version: 1\ncustom: true\n"
	if err := os.WriteFile(configPath, []byte(configContents), 0o644); err != nil {
		t.Fatalf("write existing config: %v", err)
	}
	agentsPath := filepath.Join(dir, "AGENTS.md")
	agentsContents := "# Project agents\n\nKeep existing project guidance.\n"
	if err := os.WriteFile(agentsPath, []byte(agentsContents), 0o644); err != nil {
		t.Fatalf("write existing AGENTS.md: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"init", "--config", configPath, "--agents"}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}

	if got := string(mustReadFile(t, configPath)); got != configContents {
		t.Fatalf("config = %q, want unchanged %q", got, configContents)
	}

	updatedAgents := string(mustReadFile(t, agentsPath))
	if !strings.Contains(updatedAgents, agentsContents) {
		t.Fatalf("AGENTS.md = %q, want existing guidance preserved", updatedAgents)
	}
	if !strings.Contains(updatedAgents, kitoutAgentsStartMarker) {
		t.Fatalf("AGENTS.md = %q, want Kitout section appended", updatedAgents)
	}
	if !strings.Contains(stdout.String(), "Config already exists: "+configPath) {
		t.Fatalf("stdout = %q, want existing config notice", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Updated AGENTS.md: "+agentsPath) {
		t.Fatalf("stdout = %q, want updated AGENTS path %q", stdout.String(), agentsPath)
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return contents
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
	if !strings.Contains(string(contents), "#   casks:") {
		t.Fatalf("starter config missing nested brew.casks example: %q", string(contents))
	}
	if strings.Contains(string(contents), "# casks:") {
		t.Fatalf("starter config still includes top-level casks example: %q", string(contents))
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
