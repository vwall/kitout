package config

import "path/filepath"

// CurrentVersion is the first supported Kitout config schema version.
const CurrentVersion = 1

// Config is the root YAML document for a Kitout configuration.
type Config struct {
	Version int  `yaml:"version"`
	Brew    Brew `yaml:"brew,omitempty"`
	ASDF    ASDF `yaml:"asdf,omitempty"`
	// Casks is the legacy top-level cask list. Prefer Brew.Casks.
	Casks         []string       `yaml:"casks,omitempty"`
	Directories   []string       `yaml:"directories,omitempty"`
	Copies        []Copy         `yaml:"copies,omitempty"`
	Repos         []Repo         `yaml:"repos,omitempty"`
	Symlinks      []Symlink      `yaml:"symlinks,omitempty"`
	SymlinkGroups []SymlinkGroup `yaml:"symlink_groups,omitempty"`
	MacOSDefaults []MacOSDefault `yaml:"macos_defaults,omitempty"`
	LoginShell    *LoginShell    `yaml:"login_shell,omitempty"`
	Shell         []ShellCommand `yaml:"shell,omitempty"`
}

// Brew describes Homebrew taps and packages managed by Kitout.
type Brew struct {
	Taps     []string `yaml:"taps,omitempty"`
	Packages []string `yaml:"packages,omitempty"`
	Casks    []string `yaml:"casks,omitempty"`
}

// HomebrewCasks returns the active cask list. The top-level casks field is
// kept for schema version 1 compatibility; validation rejects using both forms.
func (cfg Config) HomebrewCasks() []string {
	if len(cfg.Brew.Casks) > 0 {
		return cfg.Brew.Casks
	}
	return cfg.Casks
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
