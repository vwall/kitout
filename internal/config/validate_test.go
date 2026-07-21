package config

import (
	"errors"
	"strings"
	"testing"
)

const wantLoginShellPathValidationMessage = "must be an absolute path or homebrew:<binary> without control characters"

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
		Brew:    Brew{Taps: []string{""}, Packages: []string{""}, Casks: []string{""}},
		ASDF: ASDF{
			Plugins: []ASDFPlugin{{Versions: []string{""}}},
			ToolVersions: []ASDFToolVersion{
				{},
			},
		},
		Casks:         caskList("ghostty"),
		Directories:   []string{""},
		Repos:         []Repo{{}},
		Copies:        []Copy{{}},
		Symlinks:      []Symlink{{}},
		SymlinkGroups: []SymlinkGroup{{Paths: []string{""}}},
		MacOSDefaults: []MacOSDefault{{}},
		Security: Security{
			FileVault: &RequiredSetting{},
			Firewall:  &Firewall{},
		},
		System: System{
			XcodeCommandLineTools: &RequiredSetting{},
			Rosetta:               &RequiredSetting{},
		},
		SSH:        SSH{Keys: []SSHKey{{}}},
		LoginShell: &LoginShell{},
		Shell:      []ShellCommand{{}},
	}

	err := Validate(cfg)

	for _, field := range []string{
		"brew.taps[0]",
		"brew.packages[0]",
		"brew.casks[0]",
		"asdf.plugins[0].name",
		"asdf.plugins[0].url",
		"asdf.plugins[0].versions[0]",
		"asdf.tool_versions[0].path",
		"asdf.tool_versions[0].tools",
		"directories[0]",
		"repos[0].path",
		"repos[0].url",
		"copies[0].source",
		"copies[0].target",
		"symlinks[0].source",
		"symlinks[0].target",
		"symlink_groups[0].source_root",
		"symlink_groups[0].target_root",
		"symlink_groups[0].paths[0]",
		"macos_defaults[0].domain",
		"macos_defaults[0].key",
		"macos_defaults[0].type",
		"macos_defaults[0].value",
		"security.filevault.required",
		"security.firewall.enabled",
		"system.xcode_command_line_tools.required",
		"system.rosetta.required",
		"ssh.keys[0].path",
		"ssh.keys[0].type",
		"login_shell.path",
		"shell[0].name",
		"shell[0].command",
	} {
		assertValidationError(t, err, field, "is required")
	}
	assertValidationError(t, err, "casks", "is not supported; move entries under brew.casks")
}

func TestValidateRejectsDuplicateResources(t *testing.T) {
	cfg := Config{
		Version: CurrentVersion,
		Brew:    Brew{Taps: []string{"vwall/kitout", "vwall/kitout"}, Packages: []string{"git", "git"}, Casks: []string{"ghostty", "ghostty"}},
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
		Directories: []string{"~/code", "~/code"},
		Repos: []Repo{
			{Path: "~/code/kitout", URL: "git@github.com:vwall/kitout.git"},
			{Path: "~/code/kitout", URL: "git@github.com:vwall/kitout.git"},
		},
		Copies: []Copy{
			{Source: "~/dotfiles/codex/skills/nuxt-practices", Target: "~/.codex/skills/nuxt-practices"},
			{Source: "~/dotfiles/codex/skills/rails-practices", Target: "~/.codex/skills/nuxt-practices"},
		},
		Symlinks: []Symlink{
			{Source: "~/dotfiles/zshrc", Target: "~/.zshrc"},
			{Source: "~/dotfiles/zshrc", Target: "~/.zshrc"},
		},
		SymlinkGroups: []SymlinkGroup{
			{SourceRoot: "~/dotfiles/home", TargetRoot: "~", TargetPrefix: ".", Paths: []string{"zshrc"}},
			{SourceRoot: "~/dotfiles/home", TargetRoot: "~", TargetPrefix: ".", Paths: []string{"gitconfig", "gitconfig"}},
		},
		MacOSDefaults: []MacOSDefault{
			{Domain: "NSGlobalDomain", Key: "AppleShowAllExtensions", Type: "bool", Value: true},
			{Domain: "NSGlobalDomain", Key: "AppleShowAllExtensions", Type: "bool", Value: false},
		},
		SSH: SSH{Keys: []SSHKey{
			{Path: "~/.ssh/id_ed25519", Type: "ed25519"},
			{Path: "~/.ssh/id_ed25519", Type: "ed25519"},
		}},
		Shell: []ShellCommand{
			{Name: "Enable Corepack", Command: "corepack enable"},
			{Name: "Enable Corepack", Command: "corepack enable"},
		},
	}

	err := Validate(cfg)

	for _, field := range []string{
		"brew.taps[1]",
		"brew.packages[1]",
		"brew.casks[1]",
		"asdf.plugins[0].versions[1]",
		"asdf.plugins[1].name",
		"asdf.tool_versions[1].path",
		"directories[1]",
		"repos[1].path",
		"copies[1].target",
		"symlinks[1].target",
		"symlink_groups[0].paths[0]",
		"symlink_groups[1].paths[1]",
		"macos_defaults[1]",
		"ssh.keys[1].path",
		"shell[1].name",
	} {
		assertValidationErrorContains(t, err, field, "duplicates")
	}
}

func TestValidateRejectsTopLevelCasks(t *testing.T) {
	cfg := Config{
		Version: CurrentVersion,
		Casks:   caskList("ghostty", "ghostty"),
	}

	err := Validate(cfg)

	assertValidationError(t, err, "casks", "is not supported; move entries under brew.casks")
}

func TestValidateRejectsEmptyTopLevelCasks(t *testing.T) {
	cfg := Config{
		Version: CurrentVersion,
		Casks:   caskList(),
	}

	err := Validate(cfg)

	assertValidationError(t, err, "casks", "is not supported; move entries under brew.casks")
}

func TestValidateRejectsPresentTopLevelCasksWithoutDecodedValue(t *testing.T) {
	cfg := Config{
		Version:          CurrentVersion,
		topLevelCasksSet: true,
	}

	err := Validate(cfg)

	assertValidationError(t, err, "casks", "is not supported; move entries under brew.casks")
}

func TestValidateRejectsTopLevelCasksEvenWhenBrewCasksIsSet(t *testing.T) {
	cfg := Config{
		Version: CurrentVersion,
		Brew:    Brew{Casks: []string{"ghostty"}},
		Casks:   caskList("visual-studio-code"),
	}

	err := Validate(cfg)

	assertValidationError(t, err, "casks", "is not supported; move entries under brew.casks")
}

func TestWarningsDoNotReportRejectedTopLevelCasks(t *testing.T) {
	cfg := Config{
		Version: CurrentVersion,
		Casks:   caskList("ghostty"),
	}

	warnings := Warnings(cfg)

	if len(warnings) != 0 {
		t.Fatalf("len(Warnings) = %d, want 0: %#v", len(warnings), warnings)
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

func TestValidateRejectsSymlinkGroupTargetPrefixThatEscapesTargetRoot(t *testing.T) {
	for _, prefix := range []string{"../", "/tmp/"} {
		t.Run(prefix, func(t *testing.T) {
			cfg := Config{
				Version: CurrentVersion,
				SymlinkGroups: []SymlinkGroup{
					{SourceRoot: "~/dotfiles/home", TargetRoot: "~", TargetPrefix: prefix, Paths: []string{"gitconfig"}},
				},
			}

			err := Validate(cfg)

			assertValidationError(t, err, "symlink_groups[0].target_prefix", "must produce relative paths below target_root")
		})
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

func TestValidateAcceptsLoginShellPaths(t *testing.T) {
	for _, path := range []string{"/opt/homebrew/bin/fish", "homebrew:fish"} {
		cfg := Config{
			Version:    CurrentVersion,
			LoginShell: &LoginShell{Path: path, AddToEtcShells: true},
		}

		if err := Validate(cfg); err != nil {
			t.Fatalf("Validate(%q) returned error: %v", path, err)
		}
	}
}

func TestValidateRejectsDisabledRequiredSettings(t *testing.T) {
	cfg := Config{
		Version: CurrentVersion,
		Security: Security{
			FileVault: &RequiredSetting{Required: boolRef(false)},
		},
		System: System{
			XcodeCommandLineTools: &RequiredSetting{Required: boolRef(false)},
			Rosetta:               &RequiredSetting{Required: boolRef(false)},
		},
	}

	err := Validate(cfg)

	assertValidationError(t, err, "security.filevault.required", "must be true")
	assertValidationError(t, err, "system.xcode_command_line_tools.required", "must be true")
	assertValidationError(t, err, "system.rosetta.required", "must be true")
}

func TestValidateRejectsFirewallStealthModeWhenFirewallDisabled(t *testing.T) {
	cfg := Config{
		Version: CurrentVersion,
		Security: Security{
			Firewall: &Firewall{Enabled: boolRef(false), StealthMode: boolRef(true)},
		},
	}

	err := Validate(cfg)

	assertValidationError(t, err, "security.firewall.stealth_mode", "requires security.firewall.enabled true")
}

func TestValidateRejectsUnsupportedSSHKeyType(t *testing.T) {
	cfg := Config{
		Version: CurrentVersion,
		SSH: SSH{Keys: []SSHKey{
			{Path: "~/.ssh/id_rsa", Type: "rsa"},
		}},
	}

	err := Validate(cfg)

	assertValidationError(t, err, "ssh.keys[0].type", "must be ed25519")
}

func TestValidateRejectsInvalidLoginShellPaths(t *testing.T) {
	for _, path := range []string{"fish", "~/bin/fish", "$(brew --prefix)/bin/fish", "`brew --prefix`/bin/fish", "homebrew:", "homebrew:../fish", "homebrew:fish;rm"} {
		cfg := Config{
			Version:    CurrentVersion,
			LoginShell: &LoginShell{Path: path},
		}

		err := Validate(cfg)
		assertValidationError(t, err, "login_shell.path", wantLoginShellPathValidationMessage)
	}
}

func TestValidateRejectsLoginShellPathsWithControlCharacters(t *testing.T) {
	for _, path := range []string{
		"/opt/homebrew/bin/fish\n/bin/bash",
		"/opt/homebrew/bin/fish\r/bin/bash",
		"/opt/homebrew/bin/fish\t",
		"/opt/homebrew/bin/fish\x00",
		"homebrew:fish\n",
		"homebrew:fish\r",
		"homebrew:fish\t",
	} {
		cfg := Config{
			Version:    CurrentVersion,
			LoginShell: &LoginShell{Path: path},
		}

		err := Validate(cfg)
		assertValidationError(t, err, "login_shell.path", wantLoginShellPathValidationMessage)
	}
}

func TestValidationErrorsRenderSpecificMessage(t *testing.T) {
	err := ValidationErrors{
		{Field: "symlinks[0].target", Message: "is required"},
	}

	if got, want := err.Error(), "Invalid config: symlinks[0].target is required"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func caskList(values ...string) []string {
	return values
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
