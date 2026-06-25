package resources

import (
	"testing"

	"github.com/vwall/kitout/internal/config"
)

func TestBuildCreatesResourcesInStableExecutionOrder(t *testing.T) {
	cfg := config.Config{
		Version: config.CurrentVersion,
		Brew: config.Brew{
			Taps:     []string{"vwall/kitout"},
			Packages: []string{"asdf"},
			Casks:    []string{"ghostty"},
		},
		ASDF: config.ASDF{
			Plugins: []config.ASDFPlugin{
				{Name: "ruby", URL: "https://github.com/asdf-vm/asdf-ruby.git", Versions: []string{"3.3.6"}},
			},
			ToolVersions: []config.ASDFToolVersion{
				{Path: "/Users/example/.tool-versions", Tools: map[string]string{"ruby": "3.3.6"}},
			},
		},
		Directories: []string{"/Users/example/code"},
		Repos: []config.Repo{
			{Path: "/Users/example/code/kitout", URL: "git@example.com:kitout.git", Branch: "main"},
		},
		Copies: []config.Copy{
			{Source: "/Users/example/dotfiles/codex/skills/nuxt-practices", Target: "/Users/example/.codex/skills/nuxt-practices", Replace: false},
		},
		Symlinks: []config.Symlink{
			{Source: "/Users/example/dotfiles/zshrc", Target: "/Users/example/.zshrc", Replace: true},
		},
		SymlinkGroups: []config.SymlinkGroup{
			{
				SourceRoot: "/Users/example/dotfiles/home",
				TargetRoot: "/Users/example",
				Paths:      []string{".gitconfig", ".config/ghostty"},
			},
		},
		MacOSDefaults: []config.MacOSDefault{
			{Domain: "NSGlobalDomain", Key: "AppleShowAllExtensions", Type: "bool", Value: true},
		},
		LoginShell: &config.LoginShell{Path: "homebrew:fish", AddToEtcShells: true},
		Shell: []config.ShellCommand{
			{Name: "Enable Corepack", Command: "corepack enable", When: "missing-command:pnpm"},
		},
	}

	resources := Build(cfg, &fakeRunner{})

	got := make([]string, 0, len(resources))
	for _, resource := range resources {
		got = append(got, resource.ID())
	}
	want := []string{
		"brew_tap:vwall/kitout",
		"brew:asdf",
		"asdf_plugin:ruby",
		"asdf_tool_versions:/Users/example/.tool-versions",
		"cask:ghostty",
		"directory:/Users/example/code",
		"repo:/Users/example/code/kitout",
		"copy:/Users/example/.codex/skills/nuxt-practices",
		"symlink:/Users/example/.zshrc",
		"symlink:/Users/example/.gitconfig",
		"symlink:/Users/example/.config/ghostty",
		"macos_default:NSGlobalDomain/AppleShowAllExtensions",
		"login_shell:homebrew:fish",
		"shell:Enable Corepack",
	}
	if len(got) != len(want) {
		t.Fatalf("len(resources) = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("resource[%d] = %q, want %q; all = %#v", i, got[i], want[i], got)
		}
	}
	if _, ok := resources[11].(MacOSDefaultResource); !ok {
		t.Fatalf("resource[11] = %T, want MacOSDefaultResource", resources[11])
	}
	if _, ok := resources[12].(LoginShellResource); !ok {
		t.Fatalf("resource[12] = %T, want LoginShellResource", resources[12])
	}
}

func TestBuildCreatesCaskResourcesFromLegacyTopLevelCasks(t *testing.T) {
	cfg := config.Config{
		Version: config.CurrentVersion,
		Casks:   []string{"ghostty"},
	}

	resources := Build(cfg, &fakeRunner{})

	if len(resources) != 1 {
		t.Fatalf("len(resources) = %d, want 1", len(resources))
	}
	if got := resources[0].ID(); got != "cask:ghostty" {
		t.Fatalf("resource ID = %q, want cask:ghostty", got)
	}
}
