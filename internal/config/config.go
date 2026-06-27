package config

import "path/filepath"

// CurrentVersion is the first supported Kitout config schema version.
const CurrentVersion = 1

// Config is the root YAML document for a Kitout configuration.
type Config struct {
	Version int  `yaml:"version"`
	Brew    Brew `yaml:"brew,omitempty"`
	ASDF    ASDF `yaml:"asdf,omitempty"`
	// Casks catches the removed top-level casks field so validation can report a
	// migration error. Use Brew.Casks for supported config.
	Casks         any            `yaml:"casks,omitempty"`
	Directories   []string       `yaml:"directories,omitempty"`
	Copies        []Copy         `yaml:"copies,omitempty"`
	Repos         []Repo         `yaml:"repos,omitempty"`
	Symlinks      []Symlink      `yaml:"symlinks,omitempty"`
	SymlinkGroups []SymlinkGroup `yaml:"symlink_groups,omitempty"`
	MacOSDefaults []MacOSDefault `yaml:"macos_defaults,omitempty"`
	Security      Security       `yaml:"security,omitempty"`
	System        System         `yaml:"system,omitempty"`
	SSH           SSH            `yaml:"ssh,omitempty"`
	LoginShell    *LoginShell    `yaml:"login_shell,omitempty"`
	Shell         []ShellCommand `yaml:"shell,omitempty"`

	topLevelCasksSet bool
}

// Brew describes Homebrew taps and packages managed by Kitout.
type Brew struct {
	Taps     []string `yaml:"taps,omitempty"`
	Packages []string `yaml:"packages,omitempty"`
	Casks    []string `yaml:"casks,omitempty"`
}

// ASDF describes runtimes managed by asdf.
type ASDF struct {
	Plugins      []ASDFPlugin      `yaml:"plugins,omitempty"`
	ToolVersions []ASDFToolVersion `yaml:"tool_versions,omitempty"`
}

// ASDFPlugin describes one asdf plugin and exact tool versions.
type ASDFPlugin struct {
	Name                string   `yaml:"name"`
	URL                 string   `yaml:"url"`
	UpdateBeforeInstall bool     `yaml:"update_before_install,omitempty"`
	Versions            []string `yaml:"versions,omitempty"`
}

// ASDFToolVersion describes entries Kitout should manage in a .tool-versions file.
type ASDFToolVersion struct {
	Path  string            `yaml:"path"`
	Tools map[string]string `yaml:"tools"`
}

// Repo describes a Git repository checkout.
type Repo struct {
	Path   string `yaml:"path"`
	URL    string `yaml:"url"`
	Branch string `yaml:"branch,omitempty"`
}

// Copy describes one desired file or directory copy.
type Copy struct {
	Source  string `yaml:"source"`
	Target  string `yaml:"target"`
	Replace bool   `yaml:"replace,omitempty"`
}

// Symlink describes one desired symbolic link.
type Symlink struct {
	Source  string `yaml:"source"`
	Target  string `yaml:"target"`
	Replace bool   `yaml:"replace,omitempty"`
}

// SymlinkGroup describes symlinks that share source and target roots.
type SymlinkGroup struct {
	SourceRoot string   `yaml:"source_root"`
	TargetRoot string   `yaml:"target_root"`
	Replace    bool     `yaml:"replace,omitempty"`
	Paths      []string `yaml:"paths"`
}

// ExpandedSymlinks returns explicit symlinks plus symlink_groups expanded into
// ordinary symlink entries.
func (cfg Config) ExpandedSymlinks() []Symlink {
	count := len(cfg.Symlinks)
	for _, group := range cfg.SymlinkGroups {
		count += len(group.Paths)
	}

	symlinks := make([]Symlink, 0, count)
	symlinks = append(symlinks, cfg.Symlinks...)
	for _, group := range cfg.SymlinkGroups {
		for _, path := range group.Paths {
			symlinks = append(symlinks, Symlink{
				Source:  joinSymlinkGroupPath(group.SourceRoot, path),
				Target:  joinSymlinkGroupPath(group.TargetRoot, path),
				Replace: group.Replace,
			})
		}
	}

	return symlinks
}

func joinSymlinkGroupPath(root, path string) string {
	return filepath.Clean(filepath.Join(root, path))
}

// MacOSDefault describes one macOS defaults write target.
type MacOSDefault struct {
	Domain string `yaml:"domain"`
	Key    string `yaml:"key"`
	Type   string `yaml:"type"`
	Value  any    `yaml:"value"`
}

// Security describes desired macOS security state.
type Security struct {
	FileVault *RequiredSetting `yaml:"filevault,omitempty"`
	Firewall  *Firewall        `yaml:"firewall,omitempty"`
}

// RequiredSetting describes a resource that must be present or enabled.
type RequiredSetting struct {
	Required *bool `yaml:"required,omitempty"`
}

// Firewall describes the macOS application firewall state.
type Firewall struct {
	Enabled     *bool `yaml:"enabled,omitempty"`
	StealthMode *bool `yaml:"stealth_mode,omitempty"`
}

// System describes desired macOS system prerequisites.
type System struct {
	XcodeCommandLineTools *RequiredSetting `yaml:"xcode_command_line_tools,omitempty"`
	Rosetta               *RequiredSetting `yaml:"rosetta,omitempty"`
}

// SSH describes SSH key material managed by Kitout.
type SSH struct {
	Keys []SSHKey `yaml:"keys,omitempty"`
}

// SSHKey describes one SSH keypair.
type SSHKey struct {
	Path    string `yaml:"path"`
	Type    string `yaml:"type"`
	Comment string `yaml:"comment,omitempty"`
}

// LoginShell describes the current user's desired macOS login shell.
type LoginShell struct {
	Path           string `yaml:"path"`
	AddToEtcShells bool   `yaml:"add_to_etc_shells,omitempty"`
}

// ShellCommand describes an explicitly configured shell command.
type ShellCommand struct {
	Name    string `yaml:"name"`
	Command string `yaml:"command"`
	When    string `yaml:"when,omitempty"`
}
