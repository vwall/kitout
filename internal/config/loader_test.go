package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolvePathRejectsEmptyPath(t *testing.T) {
	_, err := ResolvePath("")
	if err == nil {
		t.Fatal("ResolvePath returned nil error, want error")
	}
	if !strings.Contains(err.Error(), "path is required") {
		t.Fatalf("error = %q, want path guidance", err.Error())
	}
}

func TestResolvePathExpandsHomePaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "home", path: "~", want: home},
		{name: "home child", path: "~/kitout.yaml", want: filepath.Join(home, "kitout.yaml")},
		{name: "env", path: "$HOME/kitout.yaml", want: filepath.Join(home, "kitout.yaml")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolvePath(tt.path)
			if err != nil {
				t.Fatalf("ResolvePath(%q) returned error: %v", tt.path, err)
			}
			if got != tt.want {
				t.Fatalf("ResolvePath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestLoadFileParsesValidConfig(t *testing.T) {
	configPath := writeConfigFile(t, `version: 1

brew:
  packages:
    - git

repos:
  - path: ~/code/kitout
    url: git@github.com:vwall/kitout.git
    branch: main
`)

	loaded, err := LoadFile(configPath)
	if err != nil {
		t.Fatalf("LoadFile returned error: %v", err)
	}

	if loaded.Path != configPath {
		t.Fatalf("loaded.Path = %q, want %q", loaded.Path, configPath)
	}
	if loaded.Config.Version != CurrentVersion {
		t.Fatalf("Version = %d, want %d", loaded.Config.Version, CurrentVersion)
	}
	if got := loaded.Config.Brew.Packages[0]; got != "git" {
		t.Fatalf("first brew package = %q, want git", got)
	}
	if got := loaded.Config.Repos[0].Branch; got != "main" {
		t.Fatalf("repo branch = %q, want main", got)
	}
}

func TestLoadFileWrapsMissingFileError(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "missing.yaml")

	_, err := LoadFile(configPath)
	if err == nil {
		t.Fatal("LoadFile returned nil error, want missing-file error")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("errors.Is(err, os.ErrNotExist) = false for %v", err)
	}
	if !strings.Contains(err.Error(), "read config "+configPath) {
		t.Fatalf("error = %q, want read config path", err.Error())
	}
}

func TestLoadFileReportsInvalidYAML(t *testing.T) {
	configPath := writeConfigFile(t, "version: [\n")

	_, err := LoadFile(configPath)
	if err == nil {
		t.Fatal("LoadFile returned nil error, want parse error")
	}
	if !strings.Contains(err.Error(), "parse config "+configPath) {
		t.Fatalf("error = %q, want parse config path", err.Error())
	}
}

func TestLoadFileRejectsUnknownTopLevelFields(t *testing.T) {
	configPath := writeConfigFile(t, `version: 1
unknown: true
`)

	_, err := LoadFile(configPath)
	if err == nil {
		t.Fatal("LoadFile returned nil error, want unknown-field error")
	}
	if !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("error = %q, want unknown field name", err.Error())
	}
}

func TestLoadFileRejectsUnknownNestedResourceFields(t *testing.T) {
	configPath := writeConfigFile(t, `version: 1

repos:
  - path: ~/code/kitout
    url: git@github.com:vwall/kitout.git
    remote: origin
`)

	_, err := LoadFile(configPath)
	if err == nil {
		t.Fatal("LoadFile returned nil error, want unknown-field error")
	}
	if !strings.Contains(err.Error(), "remote") {
		t.Fatalf("error = %q, want nested field name", err.Error())
	}
}

func writeConfigFile(t *testing.T, contents string) string {
	t.Helper()

	configPath := filepath.Join(t.TempDir(), "kitout.yaml")
	if err := os.WriteFile(configPath, []byte(contents), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	return configPath
}
