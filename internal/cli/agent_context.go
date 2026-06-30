package cli

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vwall/kitout/internal/config"
)

type agentCommand struct {
	Command string `json:"command"`
	Reason  string `json:"reason"`
}

type agentResourceSummary struct {
	ResourceID string            `json:"resource_id"`
	Type       string            `json:"type"`
	Label      string            `json:"label"`
	Details    map[string]string `json:"details,omitempty"`
}

type agentContextReport struct {
	SchemaVersion    int
	ConfigPath       string
	ConfigDir        string
	ConfigWarnings   []config.ConfigWarning
	SafeCommands     []agentCommand
	RequiresApproval []agentCommand
	ManagedResources []agentResourceSummary
	Guidance         []string
}

func buildAgentContextReport(loaded config.LoadedConfig) agentContextReport {
	return agentContextReport{
		SchemaVersion:    config.CurrentVersion,
		ConfigPath:       loaded.Path,
		ConfigDir:        filepath.Dir(loaded.Path),
		ConfigWarnings:   loaded.Warnings,
		SafeCommands:     safeAgentCommands(loaded.Path),
		RequiresApproval: approvalRequiredCommands(loaded.Path),
		ManagedResources: agentResourceSummariesFromConfig(loaded.Config),
		Guidance:         agentGuidance(),
	}
}

func safeAgentCommands(configPath string) []agentCommand {
	return []agentCommand{
		{
			Command: "kitout context --config " + quoteCommandArg(configPath),
			Reason:  "Show declared resources and agent safety guidance without checking live state.",
		},
		{
			Command: "kitout status --config " + quoteCommandArg(configPath),
			Reason:  "Check current resource state without making changes.",
		},
		{
			Command: "kitout status --config " + quoteCommandArg(configPath) + " --json",
			Reason:  "Return current resource state in stable JSON.",
		},
		{
			Command: "kitout apply --config " + quoteCommandArg(configPath) + " --dry-run",
			Reason:  "Preview planned changes without making changes.",
		},
		{
			Command: "kitout apply --config " + quoteCommandArg(configPath) + " --dry-run --json",
			Reason:  "Return the dry-run plan in stable JSON.",
		},
		{
			Command: "kitout upgrade --config " + quoteCommandArg(configPath) + " --dry-run",
			Reason:  "Preview managed Homebrew formula and cask upgrades without making changes.",
		},
		{
			Command: "kitout upgrade --config " + quoteCommandArg(configPath) + " --dry-run --json",
			Reason:  "Return the managed Homebrew upgrade plan in stable JSON.",
		},
		{
			Command: "kitout doctor --config " + quoteCommandArg(configPath),
			Reason:  "Check prerequisites, config validity, and path permissions.",
		},
		{
			Command: "kitout doctor --config " + quoteCommandArg(configPath) + " --json",
			Reason:  "Return prerequisite checks in stable JSON.",
		},
		{
			Command: "kitout explain --config " + quoteCommandArg(configPath) + " <resource-id>",
			Reason:  "Explain one configured resource and its planned action.",
		},
	}
}

func approvalRequiredCommands(configPath string) []agentCommand {
	return []agentCommand{
		{
			Command: "kitout apply --config " + quoteCommandArg(configPath),
			Reason:  "Applies changes to the user's machine.",
		},
		{
			Command: "kitout apply --config " + quoteCommandArg(configPath) + " --yes",
			Reason:  "Bypasses confirmations for risky apply actions.",
		},
		{
			Command: "kitout upgrade --config " + quoteCommandArg(configPath),
			Reason:  "Upgrades managed Homebrew formulae and casks on the user's machine.",
		},
		{
			Command: "configured shell resources",
			Reason:  "Shell resources run explicit user-provided commands during apply.",
		},
	}
}

func agentGuidance() []string {
	return []string{
		"Edit files in the setup repo or declared source paths, not managed targets in $HOME.",
		"Run status and dry-run commands before recommending a real apply or upgrade.",
		"Do not run kitout apply or kitout upgrade unless the user explicitly asks for machine changes.",
		"Do not add shell resources unless the command is explicit, idempotent, and requested.",
		"Do not store secrets in Kitout config or managed dotfiles.",
	}
}

func quoteCommandArg(value string) string {
	if value == "" {
		return "''"
	}
	if strings.IndexFunc(value, func(r rune) bool {
		return !(r == '/' || r == '.' || r == '_' || r == '-' || r == ':' ||
			(r >= '0' && r <= '9') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= 'a' && r <= 'z'))
	}) == -1 {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func agentResourceSummariesFromConfig(cfg config.Config) []agentResourceSummary {
	summaries := make([]agentResourceSummary, 0)

	if requiredSettingEnabledForContext(cfg.Security.FileVault) {
		summaries = appendAgentResource(summaries, "security", "security:filevault", map[string]string{
			"name":     "filevault",
			"required": "true",
		})
	}
	if requiredSettingEnabledForContext(cfg.System.XcodeCommandLineTools) {
		summaries = appendAgentResource(summaries, "system", "system:xcode_command_line_tools", map[string]string{
			"name":     "xcode_command_line_tools",
			"required": "true",
		})
	}
	if requiredSettingEnabledForContext(cfg.System.Rosetta) {
		summaries = appendAgentResource(summaries, "system", "system:rosetta", map[string]string{
			"name":     "rosetta",
			"required": "true",
		})
	}
	for _, name := range cfg.Brew.Taps {
		summaries = appendAgentResource(summaries, "brew_tap", "brew_tap:"+name, map[string]string{"name": name})
	}
	for _, name := range cfg.Brew.Packages {
		summaries = appendAgentResource(summaries, "brew", "brew:"+name, map[string]string{"name": name})
	}
	for _, plugin := range cfg.ASDF.Plugins {
		details := map[string]string{
			"name": plugin.Name,
			"url":  plugin.URL,
		}
		if len(plugin.Versions) > 0 {
			details["versions"] = strings.Join(plugin.Versions, ",")
		}
		if plugin.UpdateBeforeInstall {
			details["update_before_install"] = "true"
		}
		summaries = appendAgentResource(summaries, "asdf_plugin", "asdf_plugin:"+plugin.Name, details)
	}
	for _, item := range cfg.ASDF.ToolVersions {
		details := map[string]string{"path": item.Path}
		toolNames := make([]string, 0, len(item.Tools))
		for tool := range item.Tools {
			toolNames = append(toolNames, tool)
		}
		sort.Strings(toolNames)
		for _, tool := range toolNames {
			details["tool."+tool] = item.Tools[tool]
		}
		summaries = appendAgentResource(summaries, "asdf_tool_versions", "asdf_tool_versions:"+item.Path, details)
	}
	for _, name := range cfg.Brew.Casks {
		summaries = appendAgentResource(summaries, "cask", "cask:"+name, map[string]string{"name": name})
	}
	for _, path := range cfg.Directories {
		summaries = appendAgentResource(summaries, "directory", "directory:"+path, map[string]string{"path": path})
	}
	for _, repo := range cfg.Repos {
		details := map[string]string{
			"path": repo.Path,
			"url":  repo.URL,
		}
		if repo.Branch != "" {
			details["branch"] = repo.Branch
		}
		summaries = appendAgentResource(summaries, "repo", "repo:"+repo.Path, details)
	}
	for _, copy := range cfg.Copies {
		summaries = appendAgentResource(summaries, "copy", "copy:"+copy.Target, copyOrLinkDetails(copy.Source, copy.Target, copy.Replace))
	}
	for _, symlink := range cfg.ExpandedSymlinks() {
		summaries = appendAgentResource(summaries, "symlink", "symlink:"+symlink.Target, copyOrLinkDetails(symlink.Source, symlink.Target, symlink.Replace))
	}
	for _, item := range cfg.MacOSDefaults {
		summaries = appendAgentResource(summaries, "macos_default", "macos_default:"+item.Domain+"/"+item.Key, map[string]string{
			"domain": item.Domain,
			"key":    item.Key,
			"type":   item.Type,
			"value":  fmt.Sprint(item.Value),
		})
	}
	if cfg.Security.Firewall != nil && cfg.Security.Firewall.Enabled != nil {
		summaries = appendAgentResource(summaries, "security", "security:firewall", map[string]string{
			"name":    "firewall",
			"enabled": fmt.Sprintf("%t", *cfg.Security.Firewall.Enabled),
		})
		if cfg.Security.Firewall.StealthMode != nil {
			summaries = appendAgentResource(summaries, "security", "security:firewall_stealth_mode", map[string]string{
				"name":    "firewall_stealth_mode",
				"enabled": fmt.Sprintf("%t", *cfg.Security.Firewall.StealthMode),
			})
		}
	}
	for _, key := range cfg.SSH.Keys {
		details := map[string]string{
			"path":        key.Path,
			"public_path": key.Path + ".pub",
			"type":        key.Type,
		}
		if key.Comment != "" {
			details["comment"] = key.Comment
		}
		summaries = appendAgentResource(summaries, "ssh_key", "ssh_key:"+key.Path, details)
	}
	if cfg.LoginShell != nil {
		summaries = appendAgentResource(summaries, "login_shell", "login_shell:"+cfg.LoginShell.Path, map[string]string{
			"path":              cfg.LoginShell.Path,
			"add_to_etc_shells": fmt.Sprintf("%t", cfg.LoginShell.AddToEtcShells),
		})
	}
	for _, command := range cfg.Shell {
		details := map[string]string{
			"name":    command.Name,
			"command": command.Command,
		}
		if command.When != "" {
			details["when"] = command.When
		}
		summaries = appendAgentResource(summaries, "shell", "shell:"+command.Name, details)
	}

	return summaries
}

func requiredSettingEnabledForContext(setting *config.RequiredSetting) bool {
	return setting != nil && setting.Required != nil && *setting.Required
}

func copyOrLinkDetails(source, target string, replace bool) map[string]string {
	return map[string]string{
		"source":  source,
		"target":  target,
		"replace": fmt.Sprintf("%t", replace),
	}
}

func appendAgentResource(summaries []agentResourceSummary, typ, id string, details map[string]string) []agentResourceSummary {
	return append(summaries, agentResourceSummary{
		ResourceID: id,
		Type:       typ,
		Label:      displayResourceLabel(typ, id, details),
		Details:    details,
	})
}
