package cli

import (
	"context"
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

func (app application) runInit(args []string, opts globalOptions) int {
	stdout, stderr := app.stdout, app.stderr
	if err := app.ctx.Err(); err != nil {
		fmt.Fprintf(stderr, "kitout init canceled: %v\n", err)
		return exitRuntimeError
	}

	force := false
	agents := false
	home := false
	noAgentsWarning := false

	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	addGlobalFlags(fs, &opts)
	fs.BoolVar(&force, "force", false, "Overwrite an existing config file")
	fs.BoolVar(&agents, "agents", false, "Create or update AGENTS.md with Kitout guidance")
	fs.BoolVar(&home, "home", false, "Create config at ~/.config/kitout/kitout.yaml")
	fs.BoolVar(&noAgentsWarning, "no-agents-warning", false, "Stop doctor from warning about a missing AGENTS.md in this repo")

	if err := fs.Parse(args); err != nil {
		return exitValidation
	}
	if agents && noAgentsWarning {
		fmt.Fprintln(stderr, "Use either --agents or --no-agents-warning, not both.")
		return exitValidation
	}
	if home && opts.configPath != "" {
		fmt.Fprintln(stderr, "Use either --home or --config, not both.")
		return exitValidation
	}

	configPath := opts.configPath
	if configPath == "" {
		if home {
			configPath = config.DefaultPath
		} else {
			configPath = config.LocalPath
		}
	}

	resolvedPath, err := config.ResolvePath(configPath)
	if err != nil {
		fmt.Fprintf(stderr, "Invalid config path: %v\n", err)
		return exitValidation
	}
	resolvedPath, err = filepath.Abs(resolvedPath)
	if err != nil {
		fmt.Fprintf(stderr, "Invalid config path: %v\n", err)
		return exitValidation
	}

	if err := writeStarterConfig(app.ctx, resolvedPath, force); err != nil {
		if errors.Is(err, os.ErrExist) {
			if agents {
				fmt.Fprintf(stdout, "Config already exists: %s\n", resolvedPath)
				return writeAgentsForInit(app.ctx, resolvedPath, stdout, stderr)
			}
			if noAgentsWarning {
				fmt.Fprintf(stdout, "Config already exists: %s\n", resolvedPath)
				return writeNoAgentsWarningPreferenceForInit(app.ctx, resolvedPath, stdout, stderr)
			}
			fmt.Fprintf(stderr, "Config already exists: %s\nUse --force to overwrite it.\n", resolvedPath)
			return exitValidation
		}

		fmt.Fprintf(stderr, "Failed to create config: %v\n", err)
		return exitRuntimeError
	}

	fmt.Fprintf(stdout, "Created config: %s\n", resolvedPath)
	if agents {
		return writeAgentsForInit(app.ctx, resolvedPath, stdout, stderr)
	}
	if noAgentsWarning {
		return writeNoAgentsWarningPreferenceForInit(app.ctx, resolvedPath, stdout, stderr)
	}
	return exitOK
}

func writeAgentsForInit(ctx context.Context, configPath string, stdout, stderr io.Writer) int {
	if err := ctx.Err(); err != nil {
		fmt.Fprintf(stderr, "kitout init canceled: %v\n", err)
		return exitRuntimeError
	}

	agentsPath, result, err := writeKitoutAgentsFile(ctx, configPath)
	if err != nil {
		fmt.Fprintf(stderr, "Failed to create AGENTS.md: %v\n", err)
		return exitRuntimeError
	}

	switch result {
	case agentsCreated:
		fmt.Fprintf(stdout, "Created AGENTS.md: %s\n", agentsPath)
	case agentsUpdated:
		fmt.Fprintf(stdout, "Updated AGENTS.md: %s\n", agentsPath)
	case agentsUnchanged:
		fmt.Fprintf(stdout, "AGENTS.md already includes Kitout guidance: %s\n", agentsPath)
	}

	return exitOK
}

func writeNoAgentsWarningPreferenceForInit(ctx context.Context, configPath string, stdout, stderr io.Writer) int {
	if err := ctx.Err(); err != nil {
		fmt.Fprintf(stderr, "kitout init canceled: %v\n", err)
		return exitRuntimeError
	}

	if _, ok := nearestGitRepo(filepath.Dir(configPath)); !ok {
		fmt.Fprintln(stdout, "No missing AGENTS.md warning applies because the config is not inside a Git repo.")
		return exitOK
	}

	preferencesPath, err := writeSuppressMissingAgentsWarningPreference(ctx, configPath)
	if err != nil {
		fmt.Fprintf(stderr, "Failed to disable AGENTS.md warning: %v\n", err)
		return exitRuntimeError
	}

	fmt.Fprintf(stdout, "Disabled missing AGENTS.md warning for this repo: %s\n", preferencesPath)
	return exitOK
}

func writeStarterConfig(ctx context.Context, path string, force bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if !force {
		if _, err := os.Stat(path); err == nil {
			return os.ErrExist
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(starterConfig), 0o644)
}
