package cli

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/vwall/kitout/internal/config"
	"github.com/vwall/kitout/internal/engine"
	"github.com/vwall/kitout/internal/platform"
	"github.com/vwall/kitout/internal/resources"
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
	configPath, err := config.SelectPath(opts.configPath)
	if err != nil {
		return renderConfigError("status", err, opts, renderer, jsonRenderer, stderr)
	}
	loaded, err := config.LoadFile(configPath)
	if err != nil {
		return renderConfigError("status", err, opts, renderer, jsonRenderer, stderr)
	}

	if !opts.json {
		renderer.renderStatusStart(loaded.Path)
	}

	resourceList := resources.Build(loaded.Config, platform.NewExecRunner())
	var observer engine.PlanObserver
	if opts.verbose && !opts.json {
		observer = renderer
	}
	plan := engine.NewPlanner().BuildWithObserver(context.Background(), resourceList, observer)

	if opts.json {
		if err := jsonRenderer.renderPlan("status", loaded.Path, plan, false); err != nil {
			fmt.Fprintf(stderr, "Failed to render JSON: %v\n", err)
			return exitRuntimeError
		}
		return statusExitCode(plan)
	}

	renderer.renderStatusPlan("", plan)
	return statusExitCode(plan)
}

func statusExitCode(plan engine.Plan) int {
	if plan.HasFailures() {
		return exitRuntimeError
	}
	if plan.HasChanges() {
		return exitChanges
	}
	return exitOK
}
