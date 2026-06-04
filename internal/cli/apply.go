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
	"github.com/vwall/kitout/internal/platform"
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
	configPath := opts.configPath
	if configPath == "" {
		configPath = config.DefaultPath
	}

	loaded, err := config.LoadFile(configPath)
	if err != nil {
		return renderConfigError("apply", err, opts, renderer, jsonRenderer, stderr)
	}

	if !opts.json {
		renderer.renderApplyPlanStart(loaded.Path, applyOpts.dryRun)
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

		renderer.renderDryRunPlan("", plan)
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

		renderer.renderStatusPlan("", plan)
		return exitRuntimeError
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
	report := engine.NewExecutor().ApplyWithObserver(context.Background(), resourceList, plan, observer)
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

	renderer.renderApplyReport(reportPath, report)
	if report.HasFailures() {
		return exitApplyFailure
	}
	return exitOK
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
