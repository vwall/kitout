package cli

import (
	"flag"
	"fmt"
	"io"

	"github.com/vwall/kitout/internal/buildinfo"
)

func runVersion(args []string, opts globalOptions, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	fs.SetOutput(stderr)
	addGlobalFlags(fs, &opts)

	if err := fs.Parse(args); err != nil {
		return exitValidation
	}

	info := buildinfo.Current()
	fmt.Fprintf(stdout, "kitout %s\ncommit %s\nbuilt %s\n", info.Version, info.Commit, info.BuildDate)
	return exitOK
}
