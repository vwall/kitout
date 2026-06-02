package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/vwall/kitout/internal/config"
)

func runStatus(args []string, opts globalOptions, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(stderr)
	addGlobalFlags(fs, &opts)

	if err := fs.Parse(args); err != nil {
		return exitValidation
	}

	configPath := opts.configPath
	if configPath == "" {
		configPath = config.DefaultPath
	}

	loaded, err := config.LoadFile(configPath)
	if err != nil {
		var validationErrors config.ValidationErrors
		var parseError config.ParseError
		if errors.As(err, &validationErrors) {
			fmt.Fprintln(stderr, validationErrors.Error())
			return exitValidation
		}
		if errors.As(err, &parseError) {
			fmt.Fprintf(stderr, "Invalid config: %v\n", parseError)
			return exitValidation
		}

		fmt.Fprintf(stderr, "Failed to load config: %v\n", err)
		return exitRuntimeError
	}

	fmt.Fprintf(stdout, "Config valid: %s\n", loaded.Path)
	fmt.Fprintln(stdout, "Status checks are not implemented yet.")
	return exitOK
}
