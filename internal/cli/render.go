package cli

import (
	"fmt"
	"io"
)

type humanRenderer struct {
	stdout io.Writer
	stderr io.Writer
	quiet  bool
}

func newHumanRenderer(stdout, stderr io.Writer, opts globalOptions) humanRenderer {
	return humanRenderer{
		stdout: stdout,
		stderr: stderr,
		quiet:  opts.quiet,
	}
}

func (r humanRenderer) renderStatusConfigValid(path string) {
	if r.quiet {
		return
	}

	fmt.Fprintf(r.stdout, "Config valid: %s\n", path)
}

func (r humanRenderer) renderStatusChecksNotImplemented() {
	if r.quiet {
		return
	}

	fmt.Fprintln(r.stdout, "Status checks are not implemented yet.")
}

func (r humanRenderer) renderInvalidConfigDetails(err error) {
	fmt.Fprintln(r.stderr, err.Error())
}

func (r humanRenderer) renderInvalidConfig(err error) {
	fmt.Fprintf(r.stderr, "Invalid config: %v\n", err)
}

func (r humanRenderer) renderConfigLoadFailure(err error) {
	fmt.Fprintf(r.stderr, "Failed to load config: %v\n", err)
}
