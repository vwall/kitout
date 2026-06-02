package cli

import (
	"errors"
	"flag"
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

	renderer := newHumanRenderer(stdout, stderr, opts)
	configPath := opts.configPath
	if configPath == "" {
		configPath = config.DefaultPath
	}

	loaded, err := config.LoadFile(configPath)
	if err != nil {
		var validationErrors config.ValidationErrors
		var parseError config.ParseError
		if errors.As(err, &validationErrors) {
			renderer.renderInvalidConfigDetails(validationErrors)
			return exitValidation
		}
		if errors.As(err, &parseError) {
			renderer.renderInvalidConfig(parseError)
			return exitValidation
		}

		renderer.renderConfigLoadFailure(err)
		return exitRuntimeError
	}

	renderer.renderStatusConfigValid(loaded.Path)
	renderer.renderStatusChecksNotImplemented()
	return exitOK
}
