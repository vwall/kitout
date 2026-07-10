package cli

import (
	"flag"
	"fmt"

	"github.com/vwall/kitout/internal/config"
	"github.com/vwall/kitout/internal/engine"
	"github.com/vwall/kitout/internal/resources"
)

func (app application) runStatus(args []string, opts globalOptions) int {
	stdout, stderr := app.stdout, app.stderr
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
		renderer.renderConfigWarnings(loaded.Warnings)
	}

	resourceList := resources.Build(loaded.Config, app.newRunner())
	var observer engine.PlanObserver
	if !opts.json {
		observer = newStatusPlanObserver(renderer, opts.verbose)
	}
	plan := engine.NewPlanner().BuildWithObserver(app.ctx, resourceList, observer)

	if opts.json {
		if err := jsonRenderer.renderPlan("status", loaded.Path, loaded.Warnings, plan, false); err != nil {
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

type statusPlanObserver struct {
	renderer             humanRenderer
	verbose              bool
	fetchedHomebrewTaps  bool
	fetchedBrewPackages  bool
	fetchedHomebrewCasks bool
}

func newStatusPlanObserver(renderer humanRenderer, verbose bool) *statusPlanObserver {
	return &statusPlanObserver{renderer: renderer, verbose: verbose}
}

func (observer *statusPlanObserver) BeforeStatus(resource engine.Resource) {
	if observer.verbose {
		observer.renderer.BeforeStatus(resource)
		return
	}

	switch resource.Type() {
	case "brew_tap":
		if !observer.fetchedHomebrewTaps {
			observer.fetchedHomebrewTaps = true
			observer.renderProgress("Fetching Homebrew tap list...")
		}
	case "brew":
		if !observer.fetchedBrewPackages {
			observer.fetchedBrewPackages = true
			observer.renderProgress("Fetching Homebrew package list...")
		}
	case "cask":
		if !observer.fetchedHomebrewCasks {
			observer.fetchedHomebrewCasks = true
			observer.renderProgress("Fetching Homebrew cask list...")
		}
	}
}

func (observer *statusPlanObserver) renderProgress(message string) {
	if observer.renderer.quiet {
		return
	}

	fmt.Fprintf(observer.renderer.stdout, "%s %s\n", observer.renderer.progressMarker(), message)
}
