package config

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
)

// ValidationError describes one specific config schema problem.
type ValidationError struct {
	Field   string
	Message string
}

const loginShellPathValidationMessage = "must be an absolute path or homebrew:<binary> without control characters"

func (err ValidationError) Error() string {
	if err.Field == "" {
		return err.Message
	}

	return err.Field + " " + err.Message
}

// ValidationErrors is a structured collection of config validation failures.
type ValidationErrors []ValidationError

func (errs ValidationErrors) Error() string {
	switch len(errs) {
	case 0:
		return "Invalid config"
	case 1:
		return "Invalid config: " + errs[0].Error()
	default:
		messages := make([]string, 0, len(errs))
		for _, err := range errs {
			messages = append(messages, err.Error())
		}
		return "Invalid config: " + strings.Join(messages, "; ")
	}
}

// ConfigWarning describes a valid but discouraged config pattern.
type ConfigWarning struct {
	Field   string
	Message string
}

// Warnings returns non-fatal diagnostics for discouraged but valid config forms.
func Warnings(cfg Config) []ConfigWarning {
	return nil
}

// Validate checks the decoded config against Kitout's documented schema.
func Validate(cfg Config) error {
	return validate(cfg, validationOptions{checkPathDuplicates: true})
}

type validationOptions struct {
	checkPathDuplicates bool
}

func validateDecodedConfig(cfg Config) error {
	return validate(cfg, validationOptions{checkPathDuplicates: false})
}

func validate(cfg Config, opts validationOptions) error {
	var errs ValidationErrors

	if cfg.Version == 0 {
		errs.add("version", "is required")
	} else if cfg.Version != CurrentVersion {
		errs.add("version", fmt.Sprintf("must be %d", CurrentVersion))
	}

	errs.requireStrings("brew.taps", cfg.Brew.Taps)
	errs.requireStrings("brew.packages", cfg.Brew.Packages)
	errs.requireStrings("brew.casks", cfg.Brew.Casks)
	errs.requireStrings("directories", cfg.Directories)

	errs.detectDuplicateStrings("brew.taps", cfg.Brew.Taps)
	errs.detectDuplicateStrings("brew.packages", cfg.Brew.Packages)
	errs.detectDuplicateStrings("brew.casks", cfg.Brew.Casks)
	if opts.checkPathDuplicates {
		errs.detectDuplicateStrings("directories", cfg.Directories)
	}
	if cfg.topLevelCasksSet || cfg.Casks != nil {
		errs.add("casks", "is not supported; move entries under brew.casks")
	}

	for i, plugin := range cfg.ASDF.Plugins {
		errs.requireString(fmt.Sprintf("asdf.plugins[%d].name", i), plugin.Name)
		errs.requireString(fmt.Sprintf("asdf.plugins[%d].url", i), plugin.URL)
		errs.requireStrings(fmt.Sprintf("asdf.plugins[%d].versions", i), plugin.Versions)
		for j, version := range plugin.Versions {
			if strings.TrimSpace(version) == "latest" {
				errs.add(fmt.Sprintf("asdf.plugins[%d].versions[%d]", i, j), "must be an exact version, not latest")
			}
		}
		errs.detectDuplicateStrings(fmt.Sprintf("asdf.plugins[%d].versions", i), plugin.Versions)
	}
	errs.detectDuplicates(asdfPluginKeys(cfg.ASDF.Plugins))

	for i, item := range cfg.ASDF.ToolVersions {
		errs.requireString(fmt.Sprintf("asdf.tool_versions[%d].path", i), item.Path)
		if len(item.Tools) == 0 {
			errs.add(fmt.Sprintf("asdf.tool_versions[%d].tools", i), "is required")
		}
		for tool, version := range item.Tools {
			field := fmt.Sprintf("asdf.tool_versions[%d].tools[%s]", i, tool)
			errs.requireString(field, version)
			if strings.TrimSpace(tool) == "" {
				errs.add(fmt.Sprintf("asdf.tool_versions[%d].tools", i), "must not include an empty tool name")
			}
			if strings.TrimSpace(version) == "latest" {
				errs.add(field, "must be an exact version, not latest")
			}
		}
	}
	if opts.checkPathDuplicates {
		errs.detectDuplicates(asdfToolVersionPathKeys(cfg.ASDF.ToolVersions))
	}

	for i, repo := range cfg.Repos {
		errs.requireString(fmt.Sprintf("repos[%d].path", i), repo.Path)
		errs.requireString(fmt.Sprintf("repos[%d].url", i), repo.URL)
	}
	if opts.checkPathDuplicates {
		errs.detectDuplicates(repoPathKeys(cfg.Repos))
	}

	for i, copy := range cfg.Copies {
		errs.requireString(fmt.Sprintf("copies[%d].source", i), copy.Source)
		errs.requireString(fmt.Sprintf("copies[%d].target", i), copy.Target)
	}
	if opts.checkPathDuplicates {
		errs.detectDuplicates(copyTargetKeys(cfg.Copies))
	}

	for i, symlink := range cfg.Symlinks {
		errs.requireString(fmt.Sprintf("symlinks[%d].source", i), symlink.Source)
		errs.requireString(fmt.Sprintf("symlinks[%d].target", i), symlink.Target)
	}
	for i, group := range cfg.SymlinkGroups {
		errs.requireString(fmt.Sprintf("symlink_groups[%d].source_root", i), group.SourceRoot)
		errs.requireString(fmt.Sprintf("symlink_groups[%d].target_root", i), group.TargetRoot)
		if len(group.Paths) == 0 {
			errs.add(fmt.Sprintf("symlink_groups[%d].paths", i), "is required")
		}
		errs.requireStrings(fmt.Sprintf("symlink_groups[%d].paths", i), group.Paths)
		for j, path := range group.Paths {
			if strings.TrimSpace(path) == "" {
				continue
			}
			if !isRelativeSubpath(path) {
				errs.add(fmt.Sprintf("symlink_groups[%d].paths[%d]", i, j), "must be a relative path below the group roots")
				continue
			}
			if !isRelativeSubpath(prefixedSymlinkGroupPath(group.TargetPrefix, path)) {
				errs.add(fmt.Sprintf("symlink_groups[%d].target_prefix", i), "must produce relative paths below target_root")
				break
			}
		}
	}
	if opts.checkPathDuplicates {
		errs.detectDuplicates(symlinkTargetKeys(cfg))
		errs.detectConflictingTargetOwners(cfg)
	}

	for i, item := range cfg.MacOSDefaults {
		errs.requireString(fmt.Sprintf("macos_defaults[%d].domain", i), item.Domain)
		errs.requireString(fmt.Sprintf("macos_defaults[%d].key", i), item.Key)
		errs.requireString(fmt.Sprintf("macos_defaults[%d].type", i), item.Type)
		if item.Value == nil {
			errs.add(fmt.Sprintf("macos_defaults[%d].value", i), "is required")
		}
		if item.Type != "" && !validMacOSDefaultType(item.Type) {
			errs.add(fmt.Sprintf("macos_defaults[%d].type", i), "must be one of bool, int, float, string")
		}
	}
	errs.detectDuplicates(macOSDefaultKeys(cfg.MacOSDefaults))

	if cfg.Security.FileVault != nil {
		errs.requireTrue("security.filevault.required", cfg.Security.FileVault.Required)
	}
	if cfg.Security.Firewall != nil {
		errs.requireBool("security.firewall.enabled", cfg.Security.Firewall.Enabled)
		if cfg.Security.Firewall.StealthMode != nil &&
			cfg.Security.Firewall.Enabled != nil &&
			!*cfg.Security.Firewall.Enabled {
			errs.add("security.firewall.stealth_mode", "requires security.firewall.enabled true")
		}
	}

	if cfg.System.XcodeCommandLineTools != nil {
		errs.requireTrue("system.xcode_command_line_tools.required", cfg.System.XcodeCommandLineTools.Required)
	}
	if cfg.System.Rosetta != nil {
		errs.requireTrue("system.rosetta.required", cfg.System.Rosetta.Required)
	}

	for i, key := range cfg.SSH.Keys {
		errs.requireString(fmt.Sprintf("ssh.keys[%d].path", i), key.Path)
		errs.requireString(fmt.Sprintf("ssh.keys[%d].type", i), key.Type)
		if strings.TrimSpace(key.Type) != "" && key.Type != "ed25519" {
			errs.add(fmt.Sprintf("ssh.keys[%d].type", i), "must be ed25519")
		}
	}
	if opts.checkPathDuplicates {
		errs.detectDuplicates(sshKeyPathKeys(cfg.SSH.Keys))
	}

	if cfg.LoginShell != nil {
		errs.requireString("login_shell.path", cfg.LoginShell.Path)
		if strings.TrimSpace(cfg.LoginShell.Path) != "" && !validLoginShellPath(cfg.LoginShell.Path) {
			errs.add("login_shell.path", loginShellPathValidationMessage)
		}
	}

	for i, command := range cfg.Shell {
		errs.requireString(fmt.Sprintf("shell[%d].name", i), command.Name)
		errs.requireString(fmt.Sprintf("shell[%d].command", i), command.Command)
	}
	errs.detectDuplicates(shellNameKeys(cfg.Shell))

	if len(errs) > 0 {
		return errs
	}

	return nil
}

func (errs *ValidationErrors) add(field, message string) {
	*errs = append(*errs, ValidationError{Field: field, Message: message})
}

func (errs *ValidationErrors) requireString(field, value string) {
	if strings.TrimSpace(value) == "" {
		errs.add(field, "is required")
	}
}

func (errs *ValidationErrors) requireBool(field string, value *bool) {
	if value == nil {
		errs.add(field, "is required")
	}
}

func (errs *ValidationErrors) requireTrue(field string, value *bool) {
	if value == nil {
		errs.add(field, "is required")
		return
	}
	if !*value {
		errs.add(field, "must be true")
	}
}

func (errs *ValidationErrors) requireStrings(field string, values []string) {
	for i, value := range values {
		errs.requireString(fmt.Sprintf("%s[%d]", field, i), value)
	}
}

func (errs *ValidationErrors) detectDuplicateStrings(field string, values []string) {
	keys := make([]duplicateKey, 0, len(values))
	for i, value := range values {
		keys = append(keys, duplicateKey{
			Field: fmt.Sprintf("%s[%d]", field, i),
			Value: value,
		})
	}
	errs.detectDuplicates(keys)
}

func (errs *ValidationErrors) detectDuplicates(keys []duplicateKey) {
	firstFields := make(map[string]string)
	for _, key := range keys {
		value := strings.TrimSpace(key.Value)
		if value == "" {
			continue
		}

		if firstField, ok := firstFields[value]; ok {
			errs.add(key.Field, fmt.Sprintf("duplicates %s (%s)", firstField, key.display()))
			continue
		}

		firstFields[value] = key.Field
	}
}

type duplicateKey struct {
	Field   string
	Value   string
	Display string
}

func (key duplicateKey) display() string {
	if key.Display != "" {
		return key.Display
	}

	return key.Value
}

// Compare exclusive writers across resource types. Directory declarations can
// coexist with managed directories, and parent/child paths are not collisions.
func (errs *ValidationErrors) detectConflictingTargetOwners(cfg Config) {
	owners := make(map[string]duplicateKey)
	for _, keys := range [][]duplicateKey{
		repoPathKeys(cfg.Repos),
		copyTargetKeys(cfg.Copies),
		symlinkTargetKeys(cfg),
		sshKeyPathKeys(cfg.SSH.Keys),
		sshPublicKeyPathKeys(cfg.SSH.Keys),
		asdfToolVersionPathKeys(cfg.ASDF.ToolVersions),
	} {
		for _, key := range keys {
			if key.Value == "" {
				continue
			}
			if owner, ok := owners[filepath.Clean(key.Value)]; ok {
				errs.add(key.Field, fmt.Sprintf("conflicts with %s: both manage target %s", owner.Field, key.Value))
			}
		}
		// Same-type duplicates retain their existing, more specific diagnostics.
		for _, key := range keys {
			if key.Value != "" {
				owners[filepath.Clean(key.Value)] = key
			}
		}
	}
}

func repoPathKeys(repos []Repo) []duplicateKey {
	keys := make([]duplicateKey, 0, len(repos))
	for i, repo := range repos {
		keys = append(keys, duplicateKey{
			Field: fmt.Sprintf("repos[%d].path", i),
			Value: repo.Path,
		})
	}
	return keys
}

func copyTargetKeys(copies []Copy) []duplicateKey {
	keys := make([]duplicateKey, 0, len(copies))
	for i, copy := range copies {
		keys = append(keys, duplicateKey{
			Field: fmt.Sprintf("copies[%d].target", i),
			Value: copy.Target,
		})
	}
	return keys
}

func asdfPluginKeys(plugins []ASDFPlugin) []duplicateKey {
	keys := make([]duplicateKey, 0, len(plugins))
	for i, plugin := range plugins {
		keys = append(keys, duplicateKey{
			Field: fmt.Sprintf("asdf.plugins[%d].name", i),
			Value: plugin.Name,
		})
	}
	return keys
}

func asdfToolVersionPathKeys(items []ASDFToolVersion) []duplicateKey {
	keys := make([]duplicateKey, 0, len(items))
	for i, item := range items {
		keys = append(keys, duplicateKey{
			Field: fmt.Sprintf("asdf.tool_versions[%d].path", i),
			Value: item.Path,
		})
	}
	return keys
}

func symlinkTargetKeys(cfg Config) []duplicateKey {
	keys := make([]duplicateKey, 0, len(cfg.Symlinks))
	for i, symlink := range cfg.Symlinks {
		keys = append(keys, duplicateKey{
			Field: fmt.Sprintf("symlinks[%d].target", i),
			Value: symlink.Target,
		})
	}
	for i, group := range cfg.SymlinkGroups {
		for j, path := range group.Paths {
			target := ""
			display := ""
			if strings.TrimSpace(group.TargetRoot) != "" && strings.TrimSpace(path) != "" {
				target = joinSymlinkGroupPath(group.TargetRoot, prefixedSymlinkGroupPath(group.TargetPrefix, path))
				display = target
			}
			keys = append(keys, duplicateKey{
				Field:   fmt.Sprintf("symlink_groups[%d].paths[%d]", i, j),
				Value:   target,
				Display: display,
			})
		}
	}
	return keys
}

func isRelativeSubpath(path string) bool {
	if filepath.IsAbs(path) {
		return false
	}

	clean := filepath.Clean(path)
	return clean != "." && clean != ".." && !strings.HasPrefix(clean, ".."+string(filepath.Separator))
}

func macOSDefaultKeys(items []MacOSDefault) []duplicateKey {
	keys := make([]duplicateKey, 0, len(items))
	for i, item := range items {
		value := ""
		display := ""
		if strings.TrimSpace(item.Domain) != "" && strings.TrimSpace(item.Key) != "" {
			value = item.Domain + "\x00" + item.Key
			display = item.Domain + "/" + item.Key
		}
		keys = append(keys, duplicateKey{
			Field:   fmt.Sprintf("macos_defaults[%d]", i),
			Value:   value,
			Display: display,
		})
	}
	return keys
}

func sshPublicKeyPathKeys(keys []SSHKey) []duplicateKey {
	paths := sshKeyPathKeys(keys)
	for i := range paths {
		if paths[i].Value != "" {
			paths[i].Value += ".pub"
			paths[i].Field += " (public key)"
		}
	}
	return paths
}

func sshKeyPathKeys(keys []SSHKey) []duplicateKey {
	duplicates := make([]duplicateKey, 0, len(keys))
	for i, key := range keys {
		duplicates = append(duplicates, duplicateKey{
			Field: fmt.Sprintf("ssh.keys[%d].path", i),
			Value: key.Path,
		})
	}
	return duplicates
}

func shellNameKeys(commands []ShellCommand) []duplicateKey {
	keys := make([]duplicateKey, 0, len(commands))
	for i, command := range commands {
		keys = append(keys, duplicateKey{
			Field: fmt.Sprintf("shell[%d].name", i),
			Value: command.Name,
		})
	}
	return keys
}

func validMacOSDefaultType(value string) bool {
	switch value {
	case "bool", "int", "float", "string":
		return true
	default:
		return false
	}
}

func validLoginShellPath(value string) bool {
	if containsControlCharacter(value) {
		return false
	}

	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "homebrew:") {
		binary := strings.TrimPrefix(value, "homebrew:")
		return validHomebrewBinary(binary)
	}

	return filepath.IsAbs(value) && !strings.ContainsAny(value, "$`")
}

func containsControlCharacter(value string) bool {
	for _, char := range value {
		if unicode.IsControl(char) {
			return true
		}
	}
	return false
}

func validHomebrewBinary(binary string) bool {
	if binary == "" || strings.Contains(binary, "/") {
		return false
	}
	for _, char := range binary {
		if char >= 'a' && char <= 'z' ||
			char >= 'A' && char <= 'Z' ||
			char >= '0' && char <= '9' ||
			strings.ContainsRune("._+-", char) {
			continue
		}
		return false
	}
	return true
}
