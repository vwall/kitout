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
		Version:       CurrentVersion,
		Brew:          Brew{Packages: []string{""}},
		Casks:         []string{""},
		Directories:   []string{""},
		Repos:         []Repo{{}},
		Symlinks:      []Symlink{{}},
		MacOSDefaults: []MacOSDefault{{}},
		Shell:         []ShellCommand{{}},
	}

	err := Validate(cfg)

	for _, field := range []string{
		"brew.packages[0]",
		"casks[0]",
		"directories[0]",
		"repos[0].path",
		"repos[0].url",
		"symlinks[0].source",
		"symlinks[0].target",
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
		Version:     CurrentVersion,
		Brew:        Brew{Packages: []string{"git", "git"}},
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
		"casks[1]",
		"directories[1]",
		"repos[1].path",
		"symlinks[1].target",
		"macos_defaults[1]",
		"shell[1].name",
	} {
		assertValidationErrorContains(t, err, field, "duplicates")
	}
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
