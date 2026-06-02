package config

// CurrentVersion is the first supported Kitout config schema version.
const CurrentVersion = 1

// Config is the root YAML document for a Kitout configuration.
type Config struct {
	Version       int            `yaml:"version"`
	Brew          Brew           `yaml:"brew,omitempty"`
	Casks         []string       `yaml:"casks,omitempty"`
	Directories   []string       `yaml:"directories,omitempty"`
	Repos         []Repo         `yaml:"repos,omitempty"`
	Symlinks      []Symlink      `yaml:"symlinks,omitempty"`
	MacOSDefaults []MacOSDefault `yaml:"macos_defaults,omitempty"`
	Shell         []ShellCommand `yaml:"shell,omitempty"`
}

// Brew describes Homebrew packages managed by Kitout.
type Brew struct {
	Packages []string `yaml:"packages,omitempty"`
}

// Repo describes a Git repository checkout.
type Repo struct {
	Path   string `yaml:"path"`
	URL    string `yaml:"url"`
	Branch string `yaml:"branch,omitempty"`
}

// Symlink describes one desired symbolic link.
type Symlink struct {
	Source  string `yaml:"source"`
	Target  string `yaml:"target"`
	Replace bool   `yaml:"replace,omitempty"`
}

// MacOSDefault describes one macOS defaults write target.
type MacOSDefault struct {
	Domain string `yaml:"domain"`
	Key    string `yaml:"key"`
	Type   string `yaml:"type"`
	Value  any    `yaml:"value"`
}

// ShellCommand describes an explicitly configured shell command.
type ShellCommand struct {
	Name    string `yaml:"name"`
	Command string `yaml:"command"`
	When    string `yaml:"when,omitempty"`
}
