package cli

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/vwall/kitout/internal/config"
	"github.com/vwall/kitout/internal/engine"
	"github.com/vwall/kitout/internal/resources"
)

type applyOptions struct {
	dryRun bool
}

func runApply(args []string, opts globalOptions, stdin io.Reader, stdout, stderr io.Writer) int {
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
	configPath, err := config.SelectPath(opts.configPath)
	if err != nil {
		return renderConfigError("apply", err, opts, renderer, jsonRenderer, stderr)
	}
	loaded, err := config.LoadFile(configPath)
	if err != nil {
		return renderConfigError("apply", err, opts, renderer, jsonRenderer, stderr)
	}

	if !opts.json {
		renderer.renderApplyPlanStart(loaded.Path, applyOpts.dryRun)
		renderer.renderConfigWarnings(loaded.Warnings)
	}

	resourceList := resources.Build(loaded.Config, newCLIExecRunner())
	var planObserver engine.PlanObserver
	if !opts.json {
		planObserver = newApplyPlanObserver(renderer, opts.verbose)
	}
	plan := engine.NewPlanner().BuildWithObserver(context.Background(), resourceList, planObserver)

	if applyOpts.dryRun {
		if opts.json {
			if err := jsonRenderer.renderPlan("apply", loaded.Path, loaded.Warnings, plan, true); err != nil {
				fmt.Fprintf(stderr, "Failed to render JSON: %v\n", err)
				return exitRuntimeError
			}
			if plan.HasFailures() {
				return exitRuntimeError
			}
			return exitOK
		}

		renderer.renderDryRunPlan("", plan)
		if plan.HasFailures() {
			return exitRuntimeError
		}
		return exitOK
	}

	riskyItems := riskyApplyItems(plan)
	if len(riskyItems) > 0 && !opts.yes {
		if err := confirmRiskyApply(stdin, stderr, riskyItems); err != nil {
			fmt.Fprintf(stderr, "%v\n", err)
			return exitValidation
		}
	}

	var observer engine.ApplyObserver
	reportPath := loaded.Path
	if !opts.json {
		reportPath = ""
		if plan.Summary.ToApply > 0 {
			renderer.renderApplyStart("")
		}
		observer = renderer
	}
	applyRunner := newCLIExecRunner()
	if verboseApplyOutputEnabled(opts, applyOpts) && plan.Summary.ToApply > 0 {
		applyRunner = newCLIVerboseExecRunner(stdout, stderr)
	}
	applyResources := resources.BuildUncached(loaded.Config, applyRunner)
	report := engine.NewExecutor().ApplyWithObserver(context.Background(), applyResources, plan, observer)
	if opts.json {
		if err := jsonRenderer.renderApplyReport(loaded.Path, loaded.Warnings, report); err != nil {
			fmt.Fprintf(stderr, "Failed to render JSON: %v\n", err)
			return exitRuntimeError
		}
		if report.HasFailures() {
			return exitApplyFailure
		}
		return exitOK
	}

	renderer.renderApplyReport(reportPath, report)
	if report.HasFailures() {
		return exitApplyFailure
	}
	return exitOK
}

func verboseApplyOutputEnabled(opts globalOptions, applyOpts applyOptions) bool {
	return opts.verbose && !opts.quiet && !opts.json && !applyOpts.dryRun
}

type applyPlanObserver struct {
	renderer              humanRenderer
	verbose               bool
	inspectedBrewTaps     bool
	inspectedBrewPackages bool
	inspectedBrewCasks    bool
}

func newApplyPlanObserver(renderer humanRenderer, verbose bool) *applyPlanObserver {
	return &applyPlanObserver{renderer: renderer, verbose: verbose}
}

func (observer *applyPlanObserver) BeforeStatus(resource engine.Resource) {
	if observer.verbose {
		observer.renderer.BeforeStatus(resource)
		return
	}

	switch resource.Type() {
	case "brew_tap":
		if !observer.inspectedBrewTaps {
			observer.inspectedBrewTaps = true
			observer.renderProgress("Inspecting Homebrew taps...")
		}
	case "brew":
		if !observer.inspectedBrewPackages {
			observer.inspectedBrewPackages = true
			observer.renderProgress("Inspecting Homebrew packages...")
		}
	case "cask":
		if !observer.inspectedBrewCasks {
			observer.inspectedBrewCasks = true
			observer.renderProgress("Inspecting Homebrew casks...")
		}
	default:
		observer.renderer.BeforeStatus(resource)
	}
}

func (observer *applyPlanObserver) renderProgress(message string) {
	if observer.renderer.quiet {
		return
	}

	fmt.Fprintf(observer.renderer.stdout, "%s %s\n", observer.renderer.progressMarker(), message)
}

func riskyApplyItems(plan engine.Plan) []engine.PlanItem {
	items := make([]engine.PlanItem, 0)
	for _, item := range plan.Items {
		if item.Action != engine.ActionApply {
			continue
		}
		switch item.Type {
		case "shell":
			items = append(items, item)
		case "login_shell":
			items = append(items, item)
		case "security":
			items = append(items, item)
		case "ssh_key":
			items = append(items, item)
		case "copy":
			if item.State == engine.StateChanged {
				items = append(items, item)
			}
		case "symlink":
			if item.State == engine.StateChanged {
				items = append(items, item)
			}
		}
	}
	return items
}

func confirmRiskyApply(stdin io.Reader, stderr io.Writer, items []engine.PlanItem) error {
	if stdin == nil {
		return errors.New("confirmation required for risky apply actions; rerun with --yes to continue")
	}

	fmt.Fprintln(stderr, "Risky apply actions require confirmation:")
	for _, item := range items {
		fmt.Fprintf(stderr, "  %s %s", item.Type, item.ResourceID)
		if command := item.Details["command"]; command != "" {
			fmt.Fprintf(stderr, " (%s)", command)
		}
		fmt.Fprintln(stderr)
	}
	fmt.Fprint(stderr, "Continue? Type yes to apply: ")

	scanner := bufio.NewScanner(stdin)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("could not read confirmation: %w", err)
		}
		return errors.New("confirmation required; no changes made")
	}
	answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
	if answer != "yes" && answer != "y" {
		return errors.New("apply aborted; no changes made")
	}
	return nil
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
