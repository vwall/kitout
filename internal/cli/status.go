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

	renderer := newHumanRenderer(stdout, stderr, opts)
	jsonRenderer := newJSONRenderer(stdout)
	configPath := opts.configPath
	if configPath == "" {
		configPath = config.DefaultPath
	}

	loaded, err := config.LoadFile(configPath)
	if err != nil {
		var validationErrors config.ValidationErrors
		var parseError config.ParseError
		if errors.As(err, &validationErrors) {
			if opts.json {
				if err := jsonRenderer.renderValidationErrors(validationErrors); err != nil {
					fmt.Fprintf(stderr, "Failed to render JSON: %v\n", err)
					return exitRuntimeError
				}
				return exitValidation
			}

			renderer.renderInvalidConfigDetails(validationErrors)
			return exitValidation
		}
		if errors.As(err, &parseError) {
			if opts.json {
				if err := jsonRenderer.renderParseError(parseError); err != nil {
					fmt.Fprintf(stderr, "Failed to render JSON: %v\n", err)
					return exitRuntimeError
				}
				return exitValidation
			}

			renderer.renderInvalidConfig(parseError)
			return exitValidation
		}

		if opts.json {
			if err := jsonRenderer.renderConfigLoadFailure(err); err != nil {
				fmt.Fprintf(stderr, "Failed to render JSON: %v\n", err)
				return exitRuntimeError
			}
			return exitRuntimeError
		}

		renderer.renderConfigLoadFailure(err)
		return exitRuntimeError
	}

	if opts.json {
		if err := jsonRenderer.renderStatusNotImplemented(loaded.Path); err != nil {
			fmt.Fprintf(stderr, "Failed to render JSON: %v\n", err)
			return exitRuntimeError
		}
		return exitOK
	}

	renderer.renderStatusConfigValid(loaded.Path)
	renderer.renderStatusChecksNotImplemented()
	return exitOK
}
