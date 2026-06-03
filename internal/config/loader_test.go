package config

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
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

func TestLoadFileResolvesRelativeDirectoryEntriesFromConfigDirectory(t *testing.T) {
	configDir := t.TempDir()
	configPath := writeConfigFileInDir(t, configDir, `version: 1

directories:
  - cache
  - var/log
`)

	loaded, err := LoadFile(configPath)
	if err != nil {
		t.Fatalf("LoadFile returned error: %v", err)
	}

	want := []string{
		filepath.Join(configDir, "cache"),
		filepath.Join(configDir, "var", "log"),
	}
	if got := loaded.Config.Directories; !slices.Equal(got, want) {
		t.Fatalf("directories = %#v, want %#v", got, want)
	}
}

func TestLoadFileResolvesRelativeASDFToolVersionsPathsFromConfigDirectory(t *testing.T) {
	configDir := t.TempDir()
	configPath := writeConfigFileInDir(t, configDir, `version: 1

asdf:
  tool_versions:
    - path: home/.tool-versions
      tools:
        ruby: 3.3.6
`)

	loaded, err := LoadFile(configPath)
	if err != nil {
		t.Fatalf("LoadFile returned error: %v", err)
	}

	want := filepath.Join(configDir, "home", ".tool-versions")
	if got := loaded.Config.ASDF.ToolVersions[0].Path; got != want {
		t.Fatalf("asdf tool_versions path = %q, want %q", got, want)
	}
}

func TestLoadFileResolvesRelativeRepoPathsFromConfigDirectory(t *testing.T) {
	configDir := t.TempDir()
	configPath := writeConfigFileInDir(t, configDir, `version: 1

repos:
  - path: repos/kitout
    url: git@github.com:vwall/kitout.git
`)

	loaded, err := LoadFile(configPath)
	if err != nil {
		t.Fatalf("LoadFile returned error: %v", err)
	}

	want := filepath.Join(configDir, "repos", "kitout")
	if got := loaded.Config.Repos[0].Path; got != want {
		t.Fatalf("repo path = %q, want %q", got, want)
	}
}

func TestLoadFileResolvesRelativeSymlinkPathsFromConfigDirectory(t *testing.T) {
	configDir := t.TempDir()
	configPath := writeConfigFileInDir(t, configDir, `version: 1

symlinks:
  - source: dotfiles/zshrc
    target: home/.zshrc
`)

	loaded, err := LoadFile(configPath)
	if err != nil {
		t.Fatalf("LoadFile returned error: %v", err)
	}

	wantSource := filepath.Join(configDir, "dotfiles", "zshrc")
	if got := loaded.Config.Symlinks[0].Source; got != wantSource {
		t.Fatalf("symlink source = %q, want %q", got, wantSource)
	}
	wantTarget := filepath.Join(configDir, "home", ".zshrc")
	if got := loaded.Config.Symlinks[0].Target; got != wantTarget {
		t.Fatalf("symlink target = %q, want %q", got, wantTarget)
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
	var parseError ParseError
	if !errors.As(err, &parseError) {
		t.Fatalf("LoadFile error = %T %[1]v, want ParseError", err)
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

func TestLoadFileValidatesConfig(t *testing.T) {
	configPath := writeConfigFile(t, `version: 1

brew:
  packages:
    - git
    - git
`)

	_, err := LoadFile(configPath)
	if err == nil {
		t.Fatal("LoadFile returned nil error, want validation error")
	}

	var validationErrors ValidationErrors
	if !errors.As(err, &validationErrors) {
		t.Fatalf("LoadFile error = %T %[1]v, want ValidationErrors", err)
	}
	if !strings.Contains(err.Error(), "brew.packages[1] duplicates brew.packages[0] (git)") {
		t.Fatalf("error = %q, want duplicate brew package guidance", err.Error())
	}
}

func writeConfigFile(t *testing.T, contents string) string {
	t.Helper()

	return writeConfigFileInDir(t, t.TempDir(), contents)
}

func writeConfigFileInDir(t *testing.T, dir, contents string) string {
	t.Helper()

	configPath := filepath.Join(dir, "kitout.yaml")
	if err := os.WriteFile(configPath, []byte(contents), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	return configPath
}
