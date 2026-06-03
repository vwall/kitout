package config

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateAcceptsCurrentVersion(t *testing.T) {
	cfg := Config{Version: CurrentVersion}

	if err := Validate(cfg); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}

func TestValidateRequiresVersion(t *testing.T) {
	err := Validate(Config{})

	assertValidationError(t, err, "version", "is required")
}

func TestValidateRejectsUnsupportedVersion(t *testing.T) {
	err := Validate(Config{Version: CurrentVersion + 1})

	assertValidationError(t, err, "version", "must be 1")
}

func TestValidateReportsStructuredRequiredFieldErrors(t *testing.T) {
	cfg := Config{
		Version: CurrentVersion,
		Brew:    Brew{Packages: []string{""}},
		ASDF: ASDF{
			Plugins: []ASDFPlugin{{Versions: []string{""}}},
			ToolVersions: []ASDFToolVersion{
				{},
			},
		},
		Casks:         []string{""},
		Directories:   []string{""},
		Repos:         []Repo{{}},
		Symlinks:      []Symlink{{}},
		SymlinkGroups: []SymlinkGroup{{Paths: []string{""}}},
		MacOSDefaults: []MacOSDefault{{}},
		Shell:         []ShellCommand{{}},
	}

	err := Validate(cfg)

	for _, field := range []string{
		"brew.packages[0]",
		"asdf.plugins[0].name",
		"asdf.plugins[0].url",
		"asdf.plugins[0].versions[0]",
		"asdf.tool_versions[0].path",
		"asdf.tool_versions[0].tools",
		"casks[0]",
		"directories[0]",
		"repos[0].path",
		"repos[0].url",
		"symlinks[0].source",
		"symlinks[0].target",
		"symlink_groups[0].source_root",
		"symlink_groups[0].target_root",
		"symlink_groups[0].paths[0]",
		"macos_defaults[0].domain",
		"macos_defaults[0].key",
		"macos_defaults[0].type",
		"macos_defaults[0].value",
		"shell[0].name",
		"shell[0].command",
	} {
		assertValidationError(t, err, field, "is required")
	}
}

func TestValidateRejectsDuplicateResources(t *testing.T) {
	cfg := Config{
		Version: CurrentVersion,
		Brew:    Brew{Packages: []string{"git", "git"}},
		ASDF: ASDF{
			Plugins: []ASDFPlugin{
				{Name: "ruby", URL: "https://github.com/asdf-vm/asdf-ruby.git", Versions: []string{"3.3.6", "3.3.6"}},
				{Name: "ruby", URL: "https://github.com/asdf-vm/asdf-ruby.git"},
			},
			ToolVersions: []ASDFToolVersion{
				{Path: "~/.tool-versions", Tools: map[string]string{"ruby": "3.3.6"}},
				{Path: "~/.tool-versions", Tools: map[string]string{"nodejs": "22.12.0"}},
			},
		},
		Casks:       []string{"ghostty", "ghostty"},
		Directories: []string{"~/code", "~/code"},
		Repos: []Repo{
			{Path: "~/code/kitout", URL: "git@github.com:vwall/kitout.git"},
			{Path: "~/code/kitout", URL: "git@github.com:vwall/kitout.git"},
		},
		Symlinks: []Symlink{
			{Source: "~/dotfiles/zshrc", Target: "~/.zshrc"},
			{Source: "~/dotfiles/zshrc", Target: "~/.zshrc"},
		},
		SymlinkGroups: []SymlinkGroup{
			{SourceRoot: "~/dotfiles/home", TargetRoot: "~", Paths: []string{".zshrc"}},
			{SourceRoot: "~/dotfiles/home", TargetRoot: "~", Paths: []string{".gitconfig", ".gitconfig"}},
		},
		MacOSDefaults: []MacOSDefault{
			{Domain: "NSGlobalDomain", Key: "AppleShowAllExtensions", Type: "bool", Value: true},
			{Domain: "NSGlobalDomain", Key: "AppleShowAllExtensions", Type: "bool", Value: false},
		},
		Shell: []ShellCommand{
			{Name: "Enable Corepack", Command: "corepack enable"},
			{Name: "Enable Corepack", Command: "corepack enable"},
		},
	}

	err := Validate(cfg)

	for _, field := range []string{
		"brew.packages[1]",
		"asdf.plugins[0].versions[1]",
		"asdf.plugins[1].name",
		"asdf.tool_versions[1].path",
		"casks[1]",
		"directories[1]",
		"repos[1].path",
		"symlinks[1].target",
		"symlink_groups[0].paths[0]",
		"symlink_groups[1].paths[1]",
		"macos_defaults[1]",
		"shell[1].name",
	} {
		assertValidationErrorContains(t, err, field, "duplicates")
	}
}

func TestValidateRequiresSymlinkGroupPaths(t *testing.T) {
	cfg := Config{
		Version: CurrentVersion,
		SymlinkGroups: []SymlinkGroup{
			{SourceRoot: "~/dotfiles/home", TargetRoot: "~"},
		},
	}

	err := Validate(cfg)

	assertValidationError(t, err, "symlink_groups[0].paths", "is required")
}

func TestValidateRejectsSymlinkGroupPathsOutsideRoots(t *testing.T) {
	cfg := Config{
		Version: CurrentVersion,
		SymlinkGroups: []SymlinkGroup{
			{SourceRoot: "~/dotfiles/home", TargetRoot: "~", Paths: []string{"/tmp/file", "../escape", "."}},
		},
	}

	err := Validate(cfg)

	for _, field := range []string{
		"symlink_groups[0].paths[0]",
		"symlink_groups[0].paths[1]",
		"symlink_groups[0].paths[2]",
	} {
		assertValidationError(t, err, field, "must be a relative path below the group roots")
	}
}

func TestValidateRejectsASDFLatestVersions(t *testing.T) {
	cfg := Config{
		Version: CurrentVersion,
		ASDF: ASDF{
			Plugins: []ASDFPlugin{
				{Name: "ruby", URL: "https://github.com/asdf-vm/asdf-ruby.git", Versions: []string{"latest"}},
			},
			ToolVersions: []ASDFToolVersion{
				{Path: "~/.tool-versions", Tools: map[string]string{"ruby": "latest"}},
			},
		},
	}

	err := Validate(cfg)

	assertValidationError(t, err, "asdf.plugins[0].versions[0]", "must be an exact version, not latest")
	assertValidationError(t, err, "asdf.tool_versions[0].tools[ruby]", "must be an exact version, not latest")
}

func TestValidateRejectsUnsupportedMacOSDefaultType(t *testing.T) {
	cfg := Config{
		Version: CurrentVersion,
		MacOSDefaults: []MacOSDefault{
			{Domain: "NSGlobalDomain", Key: "AppleShowAllExtensions", Type: "boolean", Value: true},
		},
	}

	err := Validate(cfg)

	assertValidationError(t, err, "macos_defaults[0].type", "must be one of bool, int, float, string")
}

func TestValidationErrorsRenderSpecificMessage(t *testing.T) {
	err := ValidationErrors{
		{Field: "symlinks[0].target", Message: "is required"},
	}

	if got, want := err.Error(), "Invalid config: symlinks[0].target is required"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func assertValidationError(t *testing.T, err error, field, message string) {
	t.Helper()

	assertValidationErrorMatches(t, err, field, func(got string) bool {
		return got == message
	}, message)
}

func assertValidationErrorContains(t *testing.T, err error, field, message string) {
	t.Helper()

	assertValidationErrorMatches(t, err, field, func(got string) bool {
		return strings.Contains(got, message)
	}, message)
}

func assertValidationErrorMatches(t *testing.T, err error, field string, match func(string) bool, want string) {
	t.Helper()

	var validationErrors ValidationErrors
	if !errors.As(err, &validationErrors) {
		t.Fatalf("Validate error = %T %[1]v, want ValidationErrors", err)
	}

	for _, validationError := range validationErrors {
		if validationError.Field == field && match(validationError.Message) {
			return
		}
	}

	t.Fatalf("ValidationErrors = %#v, want %s %q", validationErrors, field, want)
}
