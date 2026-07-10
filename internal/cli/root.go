package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
)

const (
	exitOK           = 0
	exitChanges      = 1
	exitValidation   = 2
	exitRuntimeError = 3
	exitApplyFailure = 4
)

type globalOptions struct {
	configPath string
	verbose    bool
	quiet      bool
	color      bool
	noColor    bool
	json       bool
	yes        bool
}

// Run executes the Kitout CLI with a background context and returns a process
// exit code.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	return RunContext(context.Background(), args, stdin, stdout, stderr)
}

// RunContext executes the Kitout CLI and propagates ctx to all command work.
// An os.File stdin is closed on cancellation to interrupt a pending prompt.
func RunContext(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	app := newApplication(ctx, stdin, stdout, stderr)
	if stdinFile, ok := stdin.(*os.File); ok {
		app.interruptStdin = func() {
			_ = stdinFile.Close()
		}
	}
	return app.run(args)
}

func (app application) run(args []string) int {
	var opts globalOptions
	fs := flag.NewFlagSet("kitout", flag.ContinueOnError)
	fs.SetOutput(app.stderr)
	addGlobalFlags(fs, &opts)

	if err := fs.Parse(args); err != nil {
		return exitValidation
	}

	remainingArgs := fs.Args()
	if len(remainingArgs) == 0 {
		printRootHelp(app.stdout)
		return exitOK
	}

	switch remainingArgs[0] {
	case "-h", "--help", "help":
		printRootHelp(app.stdout)
		return exitOK
	case "context":
		return app.runContext(remainingArgs[1:], opts)
	case "init":
		if err := app.ctx.Err(); err != nil {
			fmt.Fprintf(app.stderr, "kitout init canceled: %v\n", err)
			return exitRuntimeError
		}
		return app.runInit(remainingArgs[1:], opts)
	case "apply":
		return app.runApply(remainingArgs[1:], opts)
	case "doctor":
		return app.runDoctor(remainingArgs[1:], opts)
	case "explain":
		return app.runExplain(remainingArgs[1:], opts)
	case "status":
		return app.runStatus(remainingArgs[1:], opts)
	case "upgrade":
		return app.runUpgrade(remainingArgs[1:], opts)
	case "version":
		return app.runVersion(remainingArgs[1:], opts)
	default:
		fmt.Fprintf(app.stderr, "kitout: unknown command %q\n\n", remainingArgs[0])
		printRootHelp(app.stderr)
		return exitValidation
	}
}

func addGlobalFlags(fs *flag.FlagSet, opts *globalOptions) {
	fs.StringVar(&opts.configPath, "config", opts.configPath, "Path to config file")
	fs.BoolVar(&opts.verbose, "verbose", opts.verbose, "Show status progress and stream subprocess output during apply or upgrade")
	fs.BoolVar(&opts.quiet, "quiet", opts.quiet, "Reduce output")
	fs.BoolVar(&opts.color, "color", opts.color, "Force colored output")
	fs.BoolVar(&opts.noColor, "no-color", opts.noColor, "Disable colored output")
	fs.BoolVar(&opts.json, "json", opts.json, "Print machine-readable JSON output")
	fs.BoolVar(&opts.yes, "yes", opts.yes, "Bypass apply confirmations for risky changes")
}

func printRootHelp(w io.Writer) {
	fmt.Fprintln(w, `Kitout equips a fresh Mac with your apps, packages, repos, dotfiles, and defaults.

Usage:
  kitout <command> [flags]

Commands:
  apply    Apply missing or incorrect resources
  context  Show agent-friendly config context and safe commands
  doctor   Check local prerequisites and common problems
  explain  Explain one configured resource
  init      Create a starter config file and optional agent guidance
  status    Check configured resources
  upgrade   Upgrade outdated managed Homebrew formulae and casks
  version   Print version metadata

Examples:
  kitout init
  kitout init --home
  kitout init --agents
  kitout init --no-agents-warning
  kitout context
  kitout status
  kitout apply --dry-run
  kitout upgrade --dry-run
  kitout upgrade brew:git
  kitout doctor
  kitout explain directory:$HOME/code
  kitout status --config ./kitout.yaml
  kitout version

Global flags:
  --config PATH       Path to config file (required when both ./kitout.yaml and home config exist)
  --verbose           Show status progress and stream subprocess output during apply or upgrade
  --quiet             Reduce output
  --color             Force colored output
  --no-color          Disable colored output
  --json              Print machine-readable JSON output
  --yes               Bypass apply confirmations for risky changes`)
}
