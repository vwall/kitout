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
		{"Casks", "casks,omitempty"},
		{"Directories", "directories,omitempty"},
		{"Repos", "repos,omitempty"},
		{"Symlinks", "symlinks,omitempty"},
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
		{"Repo", reflect.TypeOf(Repo{}), "Path", "path"},
		{"Repo", reflect.TypeOf(Repo{}), "URL", "url"},
		{"Repo", reflect.TypeOf(Repo{}), "Branch", "branch,omitempty"},
		{"Symlink", reflect.TypeOf(Symlink{}), "Source", "source"},
		{"Symlink", reflect.TypeOf(Symlink{}), "Target", "target"},
		{"Symlink", reflect.TypeOf(Symlink{}), "Replace", "replace,omitempty"},
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
		Casks:       []string{"ghostty", "visual-studio-code", "1password"},
		Directories: []string{"~/code", "~/.config"},
		Repos: []Repo{
			{Path: "~/code/aubs", URL: "git@github.com:vrwaller/aubs.git", Branch: "main"},
		},
		Symlinks: []Symlink{
			{Source: "~/dotfiles/home/zshrc", Target: "~/.zshrc", Replace: false},
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
	if got, ok := cfg.MacOSDefaults[0].Value.(bool); !ok || !got {
		t.Fatalf("macOS default value = %#v, want true bool", cfg.MacOSDefaults[0].Value)
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
