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
				SourceRoot:   "/Users/example/dotfiles/home",
				TargetRoot:   "/Users/example",
				TargetPrefix: ".",
				Paths:        []string{"gitconfig", "config/ghostty"},
			},
		},
		MacOSDefaults: []config.MacOSDefault{
			{Domain: "NSGlobalDomain", Key: "AppleShowAllExtensions", Type: "bool", Value: true},
		},
		Security: config.Security{
			FileVault: &config.RequiredSetting{Required: testBoolRef(true)},
			Firewall:  &config.Firewall{Enabled: testBoolRef(true), StealthMode: testBoolRef(true)},
		},
		System: config.System{
			XcodeCommandLineTools: &config.RequiredSetting{Required: testBoolRef(true)},
			Rosetta:               &config.RequiredSetting{Required: testBoolRef(true)},
		},
		SSH: config.SSH{
			Keys: []config.SSHKey{
				{Path: "/Users/example/.ssh/id_ed25519", Type: "ed25519", Comment: "user@example.com"},
			},
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
		"security:filevault",
		"system:xcode_command_line_tools",
		"system:rosetta",
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
		"security:firewall",
		"security:firewall_stealth_mode",
		"ssh_key:/Users/example/.ssh/id_ed25519",
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
	if _, ok := resources[0].(FileVaultResource); !ok {
		t.Fatalf("resource[0] = %T, want FileVaultResource", resources[0])
	}
	if _, ok := resources[1].(XcodeCommandLineToolsResource); !ok {
		t.Fatalf("resource[1] = %T, want XcodeCommandLineToolsResource", resources[1])
	}
	if _, ok := resources[2].(RosettaResource); !ok {
		t.Fatalf("resource[2] = %T, want RosettaResource", resources[2])
	}
	if _, ok := resources[14].(MacOSDefaultResource); !ok {
		t.Fatalf("resource[14] = %T, want MacOSDefaultResource", resources[14])
	}
	if _, ok := resources[15].(FirewallResource); !ok {
		t.Fatalf("resource[15] = %T, want FirewallResource", resources[15])
	}
	if _, ok := resources[16].(FirewallStealthModeResource); !ok {
		t.Fatalf("resource[16] = %T, want FirewallStealthModeResource", resources[16])
	}
	if _, ok := resources[17].(SSHKeyResource); !ok {
		t.Fatalf("resource[17] = %T, want SSHKeyResource", resources[17])
	}
	if _, ok := resources[18].(LoginShellResource); !ok {
		t.Fatalf("resource[18] = %T, want LoginShellResource", resources[18])
	}
}

func TestBuildIgnoresTopLevelCasks(t *testing.T) {
	cfg := config.Config{
		Version: config.CurrentVersion,
		Casks:   []string{"ghostty"},
	}

	resources := Build(cfg, &fakeRunner{})

	if len(resources) != 0 {
		t.Fatalf("len(resources) = %d, want 0", len(resources))
	}
}

func testBoolRef(value bool) *bool {
	return &value
}
