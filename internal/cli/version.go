package cli

import (
	"flag"
	"fmt"

	"github.com/vwall/kitout/internal/buildinfo"
)

func (app application) runVersion(args []string, opts globalOptions) int {
	stdout, stderr := app.stdout, app.stderr
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
