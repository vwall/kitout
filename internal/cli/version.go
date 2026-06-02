package cli

import (
	"flag"
	"fmt"
	"io"
)

var (
	version = "0.1.0"
	commit  = "unknown"
	built   = "unknown"
)

func runVersion(args []string, opts globalOptions, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	fs.SetOutput(stderr)
	addGlobalFlags(fs, &opts)

	if err := fs.Parse(args); err != nil {
		return exitValidation
	}

	fmt.Fprintf(stdout, "kitout %s\ncommit %s\nbuilt %s\n", version, commit, built)
	return exitOK
}
