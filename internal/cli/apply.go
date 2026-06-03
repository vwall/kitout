package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/vwall/kitout/internal/config"
	"github.com/vwall/kitout/internal/engine"
	"github.com/vwall/kitout/internal/platform"
	"github.com/vwall/kitout/internal/resources"
)

type applyOptions struct {
	dryRun bool
}

func runApply(args []string, opts globalOptions, stdout, stderr io.Writer) int {
	var applyOpts applyOptions
	fs := flag.NewFlagSet("apply", flag.ContinueOnError)
	fs.SetOutput(stderr)
	addGlobalFlags(fs, &opts)
	fs.BoolVar(&applyOpts.dryRun, "dry-run", applyOpts.dryRun, "Show planned changes without applying them")

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
		return renderConfigError("apply", err, opts, renderer, jsonRenderer, stderr)
	}

	resourceList := resources.Build(loaded.Config, platform.NewExecRunner())
	plan := engine.NewPlanner().Build(context.Background(), resourceList)

	if applyOpts.dryRun {
		if opts.json {
			if err := jsonRenderer.renderPlan("apply", loaded.Path, plan, true); err != nil {
				fmt.Fprintf(stderr, "Failed to render JSON: %v\n", err)
				return exitRuntimeError
			}
			if plan.HasFailures() {
				return exitRuntimeError
			}
			return exitOK
		}

		renderer.renderDryRunPlan(loaded.Path, plan)
		if plan.HasFailures() {
			return exitRuntimeError
		}
		return exitOK
	}

	if plan.HasFailures() {
		if opts.json {
			if err := jsonRenderer.renderPlan("apply", loaded.Path, plan, false); err != nil {
				fmt.Fprintf(stderr, "Failed to render JSON: %v\n", err)
				return exitRuntimeError
			}
			return exitRuntimeError
		}

		renderer.renderStatusPlan(loaded.Path, plan)
		return exitRuntimeError
	}

	report := engine.NewExecutor().Apply(context.Background(), resourceList, plan)
	if opts.json {
		if err := jsonRenderer.renderApplyReport(loaded.Path, report); err != nil {
			fmt.Fprintf(stderr, "Failed to render JSON: %v\n", err)
			return exitRuntimeError
		}
		if report.HasFailures() {
			return exitApplyFailure
		}
		return exitOK
	}

	renderer.renderApplyReport(loaded.Path, report)
	if report.HasFailures() {
		return exitApplyFailure
	}
	return exitOK
}

func renderConfigError(command string, err error, opts globalOptions, renderer humanRenderer, jsonRenderer jsonRenderer, stderr io.Writer) int {
	var validationErrors config.ValidationErrors
	var parseError config.ParseError
	if errors.As(err, &validationErrors) {
		if opts.json {
			if err := jsonRenderer.renderValidationErrors(command, validationErrors); err != nil {
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
			if err := jsonRenderer.renderParseError(command, parseError); err != nil {
				fmt.Fprintf(stderr, "Failed to render JSON: %v\n", err)
				return exitRuntimeError
			}
			return exitValidation
		}

		renderer.renderInvalidConfig(parseError)
		return exitValidation
	}

	if opts.json {
		if err := jsonRenderer.renderConfigLoadFailure(command, err); err != nil {
			fmt.Fprintf(stderr, "Failed to render JSON: %v\n", err)
			return exitRuntimeError
		}
		return exitRuntimeError
	}

	renderer.renderConfigLoadFailure(err)
	return exitRuntimeError
}
