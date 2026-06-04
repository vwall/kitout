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
		{"Casks", "casks,omitempty"},
		{"Directories", "directories,omitempty"},
		{"Repos", "repos,omitempty"},
		{"Symlinks", "symlinks,omitempty"},
		{"SymlinkGroups", "symlink_groups,omitempty"},
		{"MacOSDefaults", "macos_defaults,omitempty"},
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
		{"Brew", reflect.TypeOf(Brew{}), "Packages", "packages,omitempty"},
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
		{"Symlink", reflect.TypeOf(Symlink{}), "Source", "source"},
		{"Symlink", reflect.TypeOf(Symlink{}), "Target", "target"},
		{"Symlink", reflect.TypeOf(Symlink{}), "Replace", "replace,omitempty"},
		{"SymlinkGroup", reflect.TypeOf(SymlinkGroup{}), "SourceRoot", "source_root"},
		{"SymlinkGroup", reflect.TypeOf(SymlinkGroup{}), "TargetRoot", "target_root"},
		{"SymlinkGroup", reflect.TypeOf(SymlinkGroup{}), "Replace", "replace,omitempty"},
		{"SymlinkGroup", reflect.TypeOf(SymlinkGroup{}), "Paths", "paths"},
		{"MacOSDefault", reflect.TypeOf(MacOSDefault{}), "Domain", "domain"},
		{"MacOSDefault", reflect.TypeOf(MacOSDefault{}), "Key", "key"},
		{"MacOSDefault", reflect.TypeOf(MacOSDefault{}), "Type", "type"},
		{"MacOSDefault", reflect.TypeOf(MacOSDefault{}), "Value", "value"},
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
			Packages: []string{"git", "ruby", "node", "pnpm", "gh"},
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
		Casks:       []string{"ghostty", "visual-studio-code", "1password"},
		Directories: []string{"~/code", "~/.config"},
		Repos: []Repo{
			{Path: "~/code/aubs", URL: "git@github.com:vrwaller/aubs.git", Branch: "main"},
		},
		Symlinks: []Symlink{
			{Source: "~/dotfiles/home/zshrc", Target: "~/.zshrc", Replace: false},
		},
		SymlinkGroups: []SymlinkGroup{
			{SourceRoot: "~/dotfiles/home", TargetRoot: "~", Replace: false, Paths: []string{".gitconfig"}},
		},
		MacOSDefaults: []MacOSDefault{
			{Domain: "NSGlobalDomain", Key: "AppleShowAllExtensions", Type: "bool", Value: true},
		},
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
		},
	}

	got := cfg.ExpandedSymlinks()
	want := []Symlink{
		{Source: "/dotfiles/home/zshrc", Target: "/home/.zshrc", Replace: false},
		{Source: "/dotfiles/home/.gitconfig", Target: "/home/.gitconfig", Replace: true},
		{Source: "/dotfiles/home/.config/ghostty", Target: "/home/.config/ghostty", Replace: true},
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
