package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/vwall/kitout/internal/config"
)

const starterConfig = `# Kitout starter config.
# This file is valid as written. Edit it, then run:
#   kitout status
#   kitout apply --dry-run

version: 1

directories:
  - ~/code
  - ~/.config
  - ~/.codex/skills

# Uncomment and customize the examples below when you are ready to manage more
# of your Mac. Keep placeholder paths and repo URLs out of active config.
#
# brew:
#   taps:
#     - vwall/kitout
#   packages:
#     - git
#     - gh
#   casks:
#     - visual-studio-code
#
# repos:
#   - path: ~/code/example
#     url: git@github.com:example/example.git
#     branch: main
#
# copies:
#   # Materialize Codex skills as real directories instead of symlinks.
#   - source: ./codex/skills/nuxt-practices
#     target: ~/.codex/skills/nuxt-practices
#     replace: false
#
# symlinks:
#   - source: ~/dotfiles/home/zshrc
#     target: ~/.zshrc
#     replace: false
#
# security:
#   filevault:
#     required: true
#   firewall:
#     enabled: true
#     stealth_mode: true
#
# system:
#   xcode_command_line_tools:
#     required: true
#   rosetta:
#     required: true
#
# ssh:
#   keys:
#     - path: ~/.ssh/id_ed25519
#       type: ed25519
#       comment: user@example.com
#
# login_shell:
#   path: homebrew:fish
#   add_to_etc_shells: true
`

func runInit(args []string, opts globalOptions, stdout, stderr io.Writer) int {
	force := false

	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	addGlobalFlags(fs, &opts)
	fs.BoolVar(&force, "force", false, "Overwrite an existing config file")

	if err := fs.Parse(args); err != nil {
		return exitValidation
	}

	configPath := opts.configPath
	if configPath == "" {
		configPath = config.DefaultPath
	}

	resolvedPath, err := config.ResolvePath(configPath)
	if err != nil {
		fmt.Fprintf(stderr, "Invalid config path: %v\n", err)
		return exitValidation
	}

	if err := writeStarterConfig(resolvedPath, force); err != nil {
		if errors.Is(err, os.ErrExist) {
			fmt.Fprintf(stderr, "Config already exists: %s\nUse --force to overwrite it.\n", resolvedPath)
			return exitValidation
		}

		fmt.Fprintf(stderr, "Failed to create config: %v\n", err)
		return exitRuntimeError
	}

	fmt.Fprintf(stdout, "Created config: %s\n", resolvedPath)
	return exitOK
}

func writeStarterConfig(path string, force bool) error {
	if !force {
		if _, err := os.Stat(path); err == nil {
			return os.ErrExist
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	return os.WriteFile(path, []byte(starterConfig), 0o644)
}
