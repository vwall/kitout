package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	agentsFileName                           = "AGENTS.md"
	agentGuidancePreferencesDirName          = ".kitout"
	agentGuidancePreferencesFileName         = "agent-guidance.yaml"
	suppressMissingAgentsWarningKey          = "suppress_missing_agents_warning"
	kitoutAgentsStartMarker                  = "<!-- kitout:agents:start -->"
	kitoutAgentsEndMarker                    = "<!-- kitout:agents:end -->"
	agentGuidancePreferencesHeader           = "# Kitout repo-local agent guidance preferences.\n"
	suppressMissingAgentsWarningFileContents = agentGuidancePreferencesHeader + suppressMissingAgentsWarningKey + ": true\n"
)

type agentGuidancePreferences struct {
	SuppressMissingAgentsWarning bool `yaml:"suppress_missing_agents_warning"`
}

type agentsWriteResult string

const (
	agentsCreated   agentsWriteResult = "created"
	agentsUpdated   agentsWriteResult = "updated"
	agentsUnchanged agentsWriteResult = "unchanged"
)

func writeKitoutAgentsFile(configPath string) (string, agentsWriteResult, error) {
	agentsPath := agentsPathForConfig(configPath)
	configReference, err := agentsConfigReference(configPath, agentsPath)
	if err != nil {
		return agentsPath, "", err
	}
	section := kitoutAgentsSection(configReference)

	if err := os.MkdirAll(filepath.Dir(agentsPath), 0o755); err != nil {
		return agentsPath, "", err
	}

	existing, err := os.ReadFile(agentsPath)
	if errors.Is(err, os.ErrNotExist) {
		contents := "# AGENTS.md\n\n" + section
		if err := os.WriteFile(agentsPath, []byte(contents), 0o644); err != nil {
			return agentsPath, "", err
		}
		return agentsPath, agentsCreated, nil
	}
	if err != nil {
		return agentsPath, "", err
	}

	merged, changed, err := mergeKitoutAgentsSection(string(existing), section)
	if err != nil {
		return agentsPath, "", err
	}
	if !changed {
		return agentsPath, agentsUnchanged, nil
	}
	if err := os.WriteFile(agentsPath, []byte(merged), 0o644); err != nil {
		return agentsPath, "", err
	}
	return agentsPath, agentsUpdated, nil
}

func writeSuppressMissingAgentsWarningPreference(configPath string) (string, error) {
	repoRoot, ok := nearestGitRepo(filepath.Dir(configPath))
	if !ok {
		return "", fmt.Errorf("agent guidance warning preferences require a config inside a Git repo")
	}

	preferencesPath := agentGuidancePreferencesPath(repoRoot)
	if err := os.MkdirAll(filepath.Dir(preferencesPath), 0o755); err != nil {
		return preferencesPath, err
	}

	existing, err := os.ReadFile(preferencesPath)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(preferencesPath, []byte(suppressMissingAgentsWarningFileContents), 0o644); err != nil {
			return preferencesPath, err
		}
		return preferencesPath, nil
	}
	if err != nil {
		return preferencesPath, err
	}

	var preferences agentGuidancePreferences
	if err := yaml.Unmarshal(existing, &preferences); err != nil {
		return preferencesPath, err
	}
	if preferences.SuppressMissingAgentsWarning {
		return preferencesPath, nil
	}
	if err := os.WriteFile(preferencesPath, []byte(mergeSuppressMissingAgentsWarningPreference(string(existing))), 0o644); err != nil {
		return preferencesPath, err
	}
	return preferencesPath, nil
}

func missingAgentsWarningSuppressed(repoRoot string) (bool, string, error) {
	preferencesPath := agentGuidancePreferencesPath(repoRoot)
	contents, err := os.ReadFile(preferencesPath)
	if errors.Is(err, os.ErrNotExist) {
		return false, preferencesPath, nil
	}
	if err != nil {
		return false, preferencesPath, err
	}

	var preferences agentGuidancePreferences
	if err := yaml.Unmarshal(contents, &preferences); err != nil {
		return false, preferencesPath, err
	}
	return preferences.SuppressMissingAgentsWarning, preferencesPath, nil
}

func agentGuidancePreferencesPath(repoRoot string) string {
	return filepath.Join(repoRoot, agentGuidancePreferencesDirName, agentGuidancePreferencesFileName)
}

func mergeSuppressMissingAgentsWarningPreference(existing string) string {
	trimmed := strings.TrimRight(existing, "\n")
	if strings.TrimSpace(trimmed) == "" {
		return suppressMissingAgentsWarningFileContents
	}

	lines := strings.Split(trimmed, "\n")
	for index, line := range lines {
		key, _, ok := strings.Cut(strings.TrimSpace(line), ":")
		if ok && key == suppressMissingAgentsWarningKey {
			lines[index] = suppressMissingAgentsWarningKey + ": true"
			return strings.Join(lines, "\n") + "\n"
		}
	}

	return strings.Join(append(lines, suppressMissingAgentsWarningKey+": true"), "\n") + "\n"
}

func agentsPathForConfig(configPath string) string {
	configDir := filepath.Dir(configPath)
	if repoRoot, ok := nearestGitRepo(configDir); ok {
		return filepath.Join(repoRoot, agentsFileName)
	}
	return filepath.Join(configDir, agentsFileName)
}

func agentsConfigReference(configPath, agentsPath string) (string, error) {
	relativePath, err := filepath.Rel(filepath.Dir(agentsPath), configPath)
	if err != nil {
		return "", fmt.Errorf("resolve config path relative to %s: %w", agentsFileName, err)
	}
	if relativePath == "." {
		return "./" + filepath.Base(configPath), nil
	}
	if strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) || relativePath == ".." {
		return configPath, nil
	}
	if strings.HasPrefix(relativePath, ".") {
		return relativePath, nil
	}
	return "." + string(filepath.Separator) + relativePath, nil
}

func nearestGitRepo(startDir string) (string, bool) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", false
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, true
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func mergeKitoutAgentsSection(existing, section string) (string, bool, error) {
	start := strings.Index(existing, kitoutAgentsStartMarker)
	end := strings.Index(existing, kitoutAgentsEndMarker)

	switch {
	case start == -1 && end == -1:
		trimmed := strings.TrimRight(existing, "\n")
		if strings.TrimSpace(trimmed) == "" {
			return "# AGENTS.md\n\n" + section, true, nil
		}
		return trimmed + "\n\n" + section, true, nil
	case start == -1 || end == -1 || end < start:
		return "", false, fmt.Errorf("%s contains partial Kitout markers; edit the Kitout section manually", agentsFileName)
	}

	end += len(kitoutAgentsEndMarker)
	merged := existing[:start] + section + existing[end:]
	if merged == existing {
		return existing, false, nil
	}
	return merged, true, nil
}

func kitoutAgentsSection(configReference string) string {
	configArg := quoteCommandArg(configReference)
	return fmt.Sprintf(`%s
## Kitout

This repository contains Kitout configuration for setting up a Mac.

Kitout config is declarative desired state. Prefer editing %s and managed source files over changing generated targets in $HOME directly.

Common safe commands:

`+"```sh"+`
kitout context --config %s
kitout status --config %s
kitout apply --config %s --dry-run
kitout doctor --config %s
`+"```"+`

Only run this after reviewing the dry run and confirming the user wants machine changes:

`+"```sh"+`
kitout apply --config %s
`+"```"+`

When editing this repo:

- Prefer Kitout resources over ad hoc shell scripts.
- Do not add secrets to Kitout config or managed dotfiles.
- Do not add shell commands unless they are explicit, idempotent, and requested.
- Do not assume Linux or Windows support.
- Keep changes idempotent.
- Run `+"`kitout status`"+` and `+"`kitout apply --dry-run`"+` before recommending a real apply.
- Do not overwrite user files unless the config explicitly allows it.
- Use `+"`kitout --help`"+` and `+"`kitout <command> --help`"+` for the full command reference.

%s
`, kitoutAgentsStartMarker, configArg, configArg, configArg, configArg, configArg, configArg, kitoutAgentsEndMarker)
}
