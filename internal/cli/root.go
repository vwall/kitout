package cli

import (
	"flag"
	"fmt"
	"io"
)

const (
	exitOK           = 0
	exitChanges      = 1
	exitValidation   = 2
	exitRuntimeError = 3
)

type globalOptions struct {
	configPath string
	verbose    bool
	quiet      bool
	noColor    bool
	json       bool
	yes        bool
}

// Run executes the Kitout CLI and returns a process exit code.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var opts globalOptions
	fs := flag.NewFlagSet("kitout", flag.ContinueOnError)
	fs.SetOutput(stderr)
	addGlobalFlags(fs, &opts)

	if err := fs.Parse(args); err != nil {
		return exitValidation
	}

	remainingArgs := fs.Args()
	if len(remainingArgs) == 0 {
		printRootHelp(stdout)
		return exitOK
	}

	switch remainingArgs[0] {
	case "-h", "--help", "help":
		printRootHelp(stdout)
		return exitOK
	case "init":
		return runInit(remainingArgs[1:], opts, stdout, stderr)
	case "version":
		return runVersion(remainingArgs[1:], opts, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "kitout: unknown command %q\n\n", remainingArgs[0])
		printRootHelp(stderr)
		return exitValidation
	}
}

func addGlobalFlags(fs *flag.FlagSet, opts *globalOptions) {
	fs.StringVar(&opts.configPath, "config", opts.configPath, "Path to config file")
	fs.BoolVar(&opts.verbose, "verbose", opts.verbose, "Show detailed command output")
	fs.BoolVar(&opts.quiet, "quiet", opts.quiet, "Reduce output")
	fs.BoolVar(&opts.noColor, "no-color", opts.noColor, "Disable colored output")
	fs.BoolVar(&opts.json, "json", opts.json, "Print machine-readable JSON output")
	fs.BoolVar(&opts.yes, "yes", opts.yes, "Skip interactive confirmations when allowed")
}

func printRootHelp(w io.Writer) {
	fmt.Fprintln(w, `Kitout equips a fresh Mac with your apps, packages, repos, dotfiles, and defaults.

Usage:
  kitout <command> [flags]

Commands:
  init      Create a starter config file
  version   Print version metadata

Planned commands:
  status    Check configured resources
  apply     Apply missing or incorrect resources
  doctor    Check local prerequisites and common problems

Examples:
  kitout init
  kitout init --config ./kitout.yaml
  kitout version

Global flags:
  --config PATH       Path to config file
  --verbose           Show detailed command output
  --quiet             Reduce output
  --no-color          Disable colored output
  --json              Print machine-readable JSON output
  --yes               Skip interactive confirmations when allowed`)
}
