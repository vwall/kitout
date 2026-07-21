package config

import (
	"reflect"
	"testing"
)

func TestCurrentVersion(t *testing.T) {
	if CurrentVersion != 1 {
		t.Fatalf("CurrentVersion = %d, want 1", CurrentVersion)
	}
}

func TestConfigUsesDocumentedYAMLFields(t *testing.T) {
	tests := []struct {
		field string
		tag   string
	}{
		{"Version", "version"},
		{"Brew", "brew,omitempty"},
		{"ASDF", "asdf,omitempty"},
		{"Directories", "directories,omitempty"},
		{"Copies", "copies,omitempty"},
		{"Repos", "repos,omitempty"},
		{"Symlinks", "symlinks,omitempty"},
		{"SymlinkGroups", "symlink_groups,omitempty"},
		{"MacOSDefaults", "macos_defaults,omitempty"},
		{"Security", "security,omitempty"},
		{"System", "system,omitempty"},
		{"SSH", "ssh,omitempty"},
		{"LoginShell", "login_shell,omitempty"},
		{"Shell", "shell,omitempty"},
	}

	for _, tt := range tests {
		if got := yamlTagOf(t, reflect.TypeOf(Config{}), tt.field); got != tt.tag {
			t.Fatalf("Config.%s yaml tag = %q, want %q", tt.field, got, tt.tag)
		}
	}
}

func TestResourceStructsUseDocumentedYAMLFields(t *testing.T) {
	tests := []struct {
		name  string
		typ   reflect.Type
		field string
		tag   string
	}{
		{"Brew", reflect.TypeOf(Brew{}), "Casks", "casks,omitempty"},
		{"Brew", reflect.TypeOf(Brew{}), "Packages", "packages,omitempty"},
		{"Brew", reflect.TypeOf(Brew{}), "Taps", "taps,omitempty"},
		{"ASDF", reflect.TypeOf(ASDF{}), "Plugins", "plugins,omitempty"},
		{"ASDF", reflect.TypeOf(ASDF{}), "ToolVersions", "tool_versions,omitempty"},
		{"ASDFPlugin", reflect.TypeOf(ASDFPlugin{}), "Name", "name"},
		{"ASDFPlugin", reflect.TypeOf(ASDFPlugin{}), "URL", "url"},
		{"ASDFPlugin", reflect.TypeOf(ASDFPlugin{}), "UpdateBeforeInstall", "update_before_install,omitempty"},
		{"ASDFPlugin", reflect.TypeOf(ASDFPlugin{}), "Versions", "versions,omitempty"},
		{"ASDFToolVersion", reflect.TypeOf(ASDFToolVersion{}), "Path", "path"},
		{"ASDFToolVersion", reflect.TypeOf(ASDFToolVersion{}), "Tools", "tools"},
		{"Repo", reflect.TypeOf(Repo{}), "Path", "path"},
		{"Repo", reflect.TypeOf(Repo{}), "URL", "url"},
		{"Repo", reflect.TypeOf(Repo{}), "Branch", "branch,omitempty"},
		{"Copy", reflect.TypeOf(Copy{}), "Source", "source"},
		{"Copy", reflect.TypeOf(Copy{}), "Target", "target"},
		{"Copy", reflect.TypeOf(Copy{}), "Replace", "replace,omitempty"},
		{"Symlink", reflect.TypeOf(Symlink{}), "Source", "source"},
		{"Symlink", reflect.TypeOf(Symlink{}), "Target", "target"},
		{"Symlink", reflect.TypeOf(Symlink{}), "Replace", "replace,omitempty"},
		{"SymlinkGroup", reflect.TypeOf(SymlinkGroup{}), "SourceRoot", "source_root"},
		{"SymlinkGroup", reflect.TypeOf(SymlinkGroup{}), "TargetRoot", "target_root"},
		{"SymlinkGroup", reflect.TypeOf(SymlinkGroup{}), "TargetPrefix", "target_prefix,omitempty"},
		{"SymlinkGroup", reflect.TypeOf(SymlinkGroup{}), "Replace", "replace,omitempty"},
		{"SymlinkGroup", reflect.TypeOf(SymlinkGroup{}), "Paths", "paths"},
		{"MacOSDefault", reflect.TypeOf(MacOSDefault{}), "Domain", "domain"},
		{"MacOSDefault", reflect.TypeOf(MacOSDefault{}), "Key", "key"},
		{"MacOSDefault", reflect.TypeOf(MacOSDefault{}), "Type", "type"},
		{"MacOSDefault", reflect.TypeOf(MacOSDefault{}), "Value", "value"},
		{"Security", reflect.TypeOf(Security{}), "FileVault", "filevault,omitempty"},
		{"Security", reflect.TypeOf(Security{}), "Firewall", "firewall,omitempty"},
		{"RequiredSetting", reflect.TypeOf(RequiredSetting{}), "Required", "required,omitempty"},
		{"Firewall", reflect.TypeOf(Firewall{}), "Enabled", "enabled,omitempty"},
		{"Firewall", reflect.TypeOf(Firewall{}), "StealthMode", "stealth_mode,omitempty"},
		{"System", reflect.TypeOf(System{}), "XcodeCommandLineTools", "xcode_command_line_tools,omitempty"},
		{"System", reflect.TypeOf(System{}), "Rosetta", "rosetta,omitempty"},
		{"SSH", reflect.TypeOf(SSH{}), "Keys", "keys,omitempty"},
		{"SSHKey", reflect.TypeOf(SSHKey{}), "Path", "path"},
		{"SSHKey", reflect.TypeOf(SSHKey{}), "Type", "type"},
		{"SSHKey", reflect.TypeOf(SSHKey{}), "Comment", "comment,omitempty"},
		{"LoginShell", reflect.TypeOf(LoginShell{}), "Path", "path"},
		{"LoginShell", reflect.TypeOf(LoginShell{}), "AddToEtcShells", "add_to_etc_shells,omitempty"},
		{"ShellCommand", reflect.TypeOf(ShellCommand{}), "Name", "name"},
		{"ShellCommand", reflect.TypeOf(ShellCommand{}), "Command", "command"},
		{"ShellCommand", reflect.TypeOf(ShellCommand{}), "When", "when,omitempty"},
	}

	for _, tt := range tests {
		if got := yamlTagOf(t, tt.typ, tt.field); got != tt.tag {
			t.Fatalf("%s.%s yaml tag = %q, want %q", tt.name, tt.field, got, tt.tag)
		}
	}
}

func TestConfigCanRepresentExampleShape(t *testing.T) {
	cfg := Config{
		Version: CurrentVersion,
		Brew: Brew{
			Taps:     []string{"vwall/kitout"},
			Packages: []string{"git", "ruby", "node", "pnpm", "gh"},
			Casks:    []string{"ghostty", "visual-studio-code", "rectangle"},
		},
		ASDF: ASDF{
			Plugins: []ASDFPlugin{
				{
					Name:                "ruby",
					URL:                 "https://github.com/asdf-vm/asdf-ruby.git",
					UpdateBeforeInstall: true,
					Versions:            []string{"3.3.6"},
				},
			},
			ToolVersions: []ASDFToolVersion{
				{Path: "~/.tool-versions", Tools: map[string]string{"ruby": "3.3.6"}},
			},
		},
		Directories: []string{"~/code", "~/.config"},
		Repos: []Repo{
			{Path: "~/code/example-project", URL: "git@github.com:example/example-project.git", Branch: "main"},
		},
		Copies: []Copy{
			{Source: "./codex/skills/nuxt-practices", Target: "~/.codex/skills/nuxt-practices", Replace: false},
		},
		Symlinks: []Symlink{
			{Source: "~/dotfiles/home/zshrc", Target: "~/.zshrc", Replace: false},
		},
		SymlinkGroups: []SymlinkGroup{
			{SourceRoot: "~/dotfiles/home", TargetRoot: "~", TargetPrefix: ".", Replace: false, Paths: []string{"gitconfig"}},
		},
		MacOSDefaults: []MacOSDefault{
			{Domain: "NSGlobalDomain", Key: "AppleShowAllExtensions", Type: "bool", Value: true},
		},
		Security: Security{
			FileVault: &RequiredSetting{Required: boolRef(true)},
			Firewall:  &Firewall{Enabled: boolRef(true), StealthMode: boolRef(true)},
		},
		System: System{
			XcodeCommandLineTools: &RequiredSetting{Required: boolRef(true)},
			Rosetta:               &RequiredSetting{Required: boolRef(true)},
		},
		SSH: SSH{
			Keys: []SSHKey{
				{Path: "~/.ssh/id_ed25519", Type: "ed25519", Comment: "user@example.com"},
			},
		},
		LoginShell: &LoginShell{Path: "homebrew:fish", AddToEtcShells: true},
		Shell: []ShellCommand{
			{Name: "Enable Corepack", Command: "corepack enable", When: "missing-command:pnpm"},
		},
	}

	if cfg.Version != 1 {
		t.Fatalf("cfg.Version = %d, want 1", cfg.Version)
	}
	if got := cfg.Brew.Packages[0]; got != "git" {
		t.Fatalf("first brew package = %q, want git", got)
	}
	if got := cfg.Brew.Casks[0]; got != "ghostty" {
		t.Fatalf("first cask = %q, want ghostty", got)
	}
	if got := cfg.Repos[0].Branch; got != "main" {
		t.Fatalf("repo branch = %q, want main", got)
	}
	if got := cfg.ASDF.Plugins[0].Versions[0]; got != "3.3.6" {
		t.Fatalf("asdf ruby version = %q, want 3.3.6", got)
	}
	if !cfg.ASDF.Plugins[0].UpdateBeforeInstall {
		t.Fatal("asdf ruby update_before_install = false, want true")
	}
	if got, ok := cfg.MacOSDefaults[0].Value.(bool); !ok || !got {
		t.Fatalf("macOS default value = %#v, want true bool", cfg.MacOSDefaults[0].Value)
	}
	if got := cfg.LoginShell.Path; got != "homebrew:fish" {
		t.Fatalf("login shell path = %q, want homebrew:fish", got)
	}
	if cfg.Security.FileVault == nil || cfg.Security.FileVault.Required == nil || !*cfg.Security.FileVault.Required {
		t.Fatalf("security.filevault.required = %#v, want true", cfg.Security.FileVault)
	}
	if cfg.Security.Firewall == nil || cfg.Security.Firewall.Enabled == nil || !*cfg.Security.Firewall.Enabled {
		t.Fatalf("security.firewall.enabled = %#v, want true", cfg.Security.Firewall)
	}
	if cfg.System.Rosetta == nil || cfg.System.Rosetta.Required == nil || !*cfg.System.Rosetta.Required {
		t.Fatalf("system.rosetta.required = %#v, want true", cfg.System.Rosetta)
	}
	if got := cfg.SSH.Keys[0].Type; got != "ed25519" {
		t.Fatalf("ssh key type = %q, want ed25519", got)
	}
}

func TestConfigExpandsSymlinkGroups(t *testing.T) {
	cfg := Config{
		Symlinks: []Symlink{
			{Source: "/dotfiles/home/zshrc", Target: "/home/.zshrc", Replace: false},
		},
		SymlinkGroups: []SymlinkGroup{
			{
				SourceRoot: "/dotfiles/home",
				TargetRoot: "/home",
				Replace:    true,
				Paths:      []string{".gitconfig", ".config/ghostty"},
			},
			{
				SourceRoot:   "/dotfiles/home",
				TargetRoot:   "/home",
				TargetPrefix: ".",
				Paths:        []string{"gitignore-global", "config/nvim"},
			},
		},
	}

	got := cfg.ExpandedSymlinks()
	want := []Symlink{
		{Source: "/dotfiles/home/zshrc", Target: "/home/.zshrc", Replace: false},
		{Source: "/dotfiles/home/.gitconfig", Target: "/home/.gitconfig", Replace: true},
		{Source: "/dotfiles/home/.config/ghostty", Target: "/home/.config/ghostty", Replace: true},
		{Source: "/dotfiles/home/gitignore-global", Target: "/home/.gitignore-global", Replace: false},
		{Source: "/dotfiles/home/config/nvim", Target: "/home/.config/nvim", Replace: false},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ExpandedSymlinks() = %#v, want %#v", got, want)
	}
}

func yamlTagOf(t *testing.T, typ reflect.Type, field string) string {
	t.Helper()

	structField, ok := typ.FieldByName(field)
	if !ok {
		t.Fatalf("%s is missing field %s", typ.Name(), field)
	}

	return structField.Tag.Get("yaml")
}

func boolRef(value bool) *bool {
	return &value
}
