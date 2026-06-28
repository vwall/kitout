package cli

import (
	"flag"
	"fmt"
	"io"

	"github.com/vwall/kitout/internal/config"
)

func runContext(args []string, opts globalOptions, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("context", flag.ContinueOnError)
	fs.SetOutput(stderr)
	addGlobalFlags(fs, &opts)

	if err := fs.Parse(args); err != nil {
		return exitValidation
	}

	renderer := newHumanRenderer(stdout, stderr, opts)
	jsonRenderer := newJSONRenderer(stdout)
	configPath, err := config.SelectPath(opts.configPath)
	if err != nil {
		return renderConfigError("context", err, opts, renderer, jsonRenderer, stderr)
	}
	loaded, err := config.LoadFile(configPath)
	if err != nil {
		return renderConfigError("context", err, opts, renderer, jsonRenderer, stderr)
	}

	report := buildAgentContextReport(loaded)
	if opts.json {
		if err := jsonRenderer.renderAgentContext(report); err != nil {
			fmt.Fprintf(stderr, "Failed to render JSON: %v\n", err)
			return exitRuntimeError
		}
		return exitOK
	}

	renderer.renderConfigWarnings(loaded.Warnings)
	renderer.renderAgentContext(report)
	return exitOK
}
