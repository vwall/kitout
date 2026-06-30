package cli

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/vwall/kitout/internal/config"
	"github.com/vwall/kitout/internal/engine"
	"github.com/vwall/kitout/internal/resources"
)

type upgradeOptions struct {
	dryRun bool
	only   string
}

type upgradeableResource interface {
	engine.Resource
	Upgrade(ctx context.Context) (engine.ApplyResult, error)
}

type upgradeApplyResource struct {
	upgradeableResource
}

type unsupportedUpgradeResource struct {
	engine.Resource
}

func (resource upgradeApplyResource) Apply(ctx context.Context) (engine.ApplyResult, error) {
	return resource.upgradeableResource.Upgrade(ctx)
}

func (resource unsupportedUpgradeResource) Apply(ctx context.Context) (engine.ApplyResult, error) {
	if err := ctx.Err(); err != nil {
		return engine.ApplyResult{
			ResourceID: resource.ID(),
			Type:       resource.Type(),
			Action:     "fail",
			Message:    "context canceled while upgrading resource",
			Details:    map[string]string{},
		}, err
	}

	err := fmt.Errorf("%s does not support upgrade", resource.ID())
	return engine.ApplyResult{
		ResourceID: resource.ID(),
		Type:       resource.Type(),
		Action:     "fail",
		Message:    "resource does not support upgrade",
		Details:    map[string]string{},
	}, err
}

func runUpgrade(args []string, opts globalOptions, stdout, stderr io.Writer) int {
	var upgradeOpts upgradeOptions
	fs := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	fs.SetOutput(stderr)
	addGlobalFlags(fs, &opts)
	fs.BoolVar(&upgradeOpts.dryRun, "dry-run", upgradeOpts.dryRun, "Show planned upgrades without applying them")
	fs.StringVar(&upgradeOpts.only, "only", upgradeOpts.only, "Limit upgrades to brew or cask")

	if err := fs.Parse(args); err != nil {
		return exitValidation
	}

	only, err := normalizeUpgradeOnly(upgradeOpts.only)
	renderer := newHumanRenderer(stdout, stderr, opts)
	jsonRenderer := newJSONRenderer(stdout)
	if err != nil {
		return renderUpgradeValidationError(err, opts, jsonRenderer, stderr)
	}

	configPath, err := config.SelectPath(opts.configPath)
	if err != nil {
		return renderConfigError("upgrade", err, opts, renderer, jsonRenderer, stderr)
	}
	loaded, err := config.LoadFile(configPath)
	if err != nil {
		return renderConfigError("upgrade", err, opts, renderer, jsonRenderer, stderr)
	}

	if !opts.json {
		renderer.renderUpgradePlanStart(loaded.Path, upgradeOpts.dryRun)
		renderer.renderConfigWarnings(loaded.Warnings)
	}

	planResources := filterUpgradeResources(resources.Build(loaded.Config, newCLIExecRunner()), only)
	var planObserver engine.PlanObserver
	if !opts.json {
		planObserver = newApplyPlanObserver(renderer, opts.verbose)
	}
	plan := buildUpgradePlan(context.Background(), planResources, planObserver)

	if upgradeOpts.dryRun {
		if opts.json {
			if err := jsonRenderer.renderPlan("upgrade", loaded.Path, loaded.Warnings, plan, true); err != nil {
				fmt.Fprintf(stderr, "Failed to render JSON: %v\n", err)
				return exitRuntimeError
			}
			if plan.HasFailures() {
				return exitRuntimeError
			}
			return exitOK
		}

		renderer.renderUpgradeDryRunPlan("", plan)
		if plan.HasFailures() {
			return exitRuntimeError
		}
		return exitOK
	}

	var observer engine.ApplyObserver
	reportPath := loaded.Path
	if !opts.json {
		reportPath = ""
		if plan.Summary.ToApply > 0 {
			renderer.renderUpgradeStart("")
		}
		observer = renderer
	}
	upgradeRunner := newCLIExecRunner()
	if verboseUpgradeOutputEnabled(opts, upgradeOpts) && plan.Summary.ToApply > 0 {
		upgradeRunner = newCLIVerboseExecRunner(stdout, stderr)
	}
	upgradeResources := wrapUpgradeResources(filterUpgradeResources(resources.BuildUncached(loaded.Config, upgradeRunner), only))
	report := engine.NewExecutor().ApplyWithObserver(context.Background(), upgradeResources, plan, observer)
	if opts.json {
		if err := jsonRenderer.renderUpgradeReport(loaded.Path, loaded.Warnings, report); err != nil {
			fmt.Fprintf(stderr, "Failed to render JSON: %v\n", err)
			return exitRuntimeError
		}
		if report.HasFailures() {
			return exitApplyFailure
		}
		return exitOK
	}

	renderer.renderUpgradeReport(reportPath, report)
	if report.HasFailures() {
		return exitApplyFailure
	}
	return exitOK
}

func normalizeUpgradeOnly(value string) (string, error) {
	switch value {
	case "", "all":
		return "", nil
	case "brew", "formula", "formulae", "formulas", "package", "packages":
		return "brew", nil
	case "cask", "casks":
		return "cask", nil
	default:
		return "", fmt.Errorf("--only must be brew or cask, got %q", value)
	}
}

func renderUpgradeValidationError(err error, opts globalOptions, jsonRenderer jsonRenderer, stderr io.Writer) int {
	if opts.json {
		if renderErr := jsonRenderer.renderValidationMessage("upgrade", err.Error()); renderErr != nil {
			fmt.Fprintf(stderr, "Failed to render JSON: %v\n", renderErr)
			return exitRuntimeError
		}
		return exitValidation
	}

	fmt.Fprintf(stderr, "kitout upgrade: %v\n", err)
	return exitValidation
}

func filterUpgradeResources(resourceList []engine.Resource, only string) []engine.Resource {
	filtered := make([]engine.Resource, 0, len(resourceList))
	for _, resource := range resourceList {
		if resource.Type() != "brew" && resource.Type() != "cask" {
			continue
		}
		if only != "" && resource.Type() != only {
			continue
		}
		filtered = append(filtered, resource)
	}
	return filtered
}

func wrapUpgradeResources(resourceList []engine.Resource) []engine.Resource {
	wrapped := make([]engine.Resource, 0, len(resourceList))
	for _, resource := range resourceList {
		upgradeable, ok := resource.(upgradeableResource)
		if !ok {
			wrapped = append(wrapped, unsupportedUpgradeResource{Resource: resource})
			continue
		}
		wrapped = append(wrapped, upgradeApplyResource{upgradeableResource: upgradeable})
	}
	return wrapped
}

func buildUpgradePlan(ctx context.Context, resourceList []engine.Resource, observer engine.PlanObserver) engine.Plan {
	statusPlan := engine.NewPlanner().BuildWithObserver(ctx, resourceList, observer)
	items := make([]engine.PlanItem, 0, len(statusPlan.Items))
	for _, item := range statusPlan.Items {
		items = append(items, upgradePlanItem(item))
	}
	return engine.NewPlanFromItems(items)
}

func upgradePlanItem(item engine.PlanItem) engine.PlanItem {
	if item.Action == engine.ActionFail {
		return item
	}

	if advisory, ok := upgradeCheckFailedAdvisory(item); ok {
		item.State = engine.StateFailed
		item.Action = engine.ActionFail
		item.Message = advisory.Message
		item.Error = advisory.Details["error"]
		item.Advisories = nil
		return item
	}

	switch item.State {
	case engine.StateSatisfied:
		advisory, ok := upgradeAvailableAdvisory(item)
		if !ok {
			item.Action = engine.ActionNoop
			return item
		}
		item.State = engine.StateChanged
		item.Action = engine.ActionApply
		item.Message = advisory.Message
		item.Advisories = nil
		return item
	case engine.StateMissing:
		item.State = engine.StateSkipped
		item.Action = engine.ActionSkip
		item.Message = missingUpgradeMessage(item)
		return item
	case engine.StateSkipped:
		item.Action = engine.ActionSkip
		return item
	case engine.StateFailed, engine.StateUnknown:
		item.Action = engine.ActionFail
		return item
	default:
		item.State = engine.StateFailed
		item.Action = engine.ActionFail
		item.Message = "cannot determine upgrade state"
		return item
	}
}

func upgradeAvailableAdvisory(item engine.PlanItem) (engine.Advisory, bool) {
	switch item.Type {
	case "brew":
		return advisoryByCode(item.Advisories, resources.HomebrewFormulaOutdatedAdvisory)
	case "cask":
		return advisoryByCode(item.Advisories, resources.HomebrewCaskOutdatedAdvisory)
	default:
		return engine.Advisory{}, false
	}
}

func upgradeCheckFailedAdvisory(item engine.PlanItem) (engine.Advisory, bool) {
	switch item.Type {
	case "brew":
		return advisoryByCode(item.Advisories, resources.HomebrewFormulaUpdateCheckFailedAdvisory)
	case "cask":
		return advisoryByCode(item.Advisories, resources.HomebrewCaskUpdateCheckFailedAdvisory)
	default:
		return engine.Advisory{}, false
	}
}

func advisoryByCode(advisories []engine.Advisory, code string) (engine.Advisory, bool) {
	for _, advisory := range advisories {
		if advisory.Code == code {
			return advisory, true
		}
	}
	return engine.Advisory{}, false
}

func missingUpgradeMessage(item engine.PlanItem) string {
	switch item.Type {
	case "brew":
		return "formula is missing; run `kitout apply` first"
	case "cask":
		return "cask is missing; run `kitout apply` first"
	default:
		return "resource is missing; run `kitout apply` first"
	}
}

func verboseUpgradeOutputEnabled(opts globalOptions, upgradeOpts upgradeOptions) bool {
	return opts.verbose && !opts.quiet && !opts.json && !upgradeOpts.dryRun
}

func (r humanRenderer) renderUpgradePlanStart(path string, dryRun bool) {
	if r.quiet {
		return
	}

	if dryRun {
		fmt.Fprintf(r.stdout, "%s Kitout is checking managed Homebrew upgrades. No changes will be made.\n", r.dryRunBadge())
	} else {
		fmt.Fprintln(r.stdout, "Kitout is planning managed Homebrew upgrades...")
	}
	fmt.Fprintf(r.stdout, "Config: %s\n\n", path)
}

func (r humanRenderer) renderUpgradeDryRunPlan(path string, plan engine.Plan) {
	if r.quiet {
		return
	}

	if path != "" {
		fmt.Fprintf(r.stdout, "Config: %s\n\n", path)
	}
	fmt.Fprintf(r.stdout, "%s Previewing managed upgrades:\n", r.dryRunBadge())
	for _, item := range plan.Items {
		switch item.Action {
		case engine.ActionApply:
			fmt.Fprintf(r.stdout, "%s %s\n", r.dryRunMarker(), r.colorize(dryRunMessage(item), ansiYellow))
		case engine.ActionFail:
			fmt.Fprintf(r.stdout, "%s Cannot upgrade %s: %s\n", r.failSymbol(), displayResourceLabel(item.Type, item.ResourceID, item.Details), item.Message)
			if item.Error != "" {
				renderIndentedDetail(r.stdout, "    ", "error", item.Error)
			}
		case engine.ActionSkip:
			fmt.Fprintf(r.stdout, "%s Skipping %s: %s\n", r.skipSymbol(), displayResourceLabel(item.Type, item.ResourceID, item.Details), item.Message)
		}
	}
	if plan.Summary.ToApply == 0 {
		fmt.Fprintln(r.stdout, "No upgrades.")
	}
	r.renderPlanAdvisories(plan)
	fmt.Fprintf(r.stdout, "\n%s No upgrades made because --dry-run was used.\n", r.dryRunBadge())
}

func (r humanRenderer) renderUpgradeStart(path string) {
	if r.quiet {
		return
	}

	if path != "" {
		fmt.Fprintf(r.stdout, "Config: %s\n\n", path)
	}
	fmt.Fprintln(r.stdout, "Upgrading managed Homebrew resources:")
}

func (r humanRenderer) renderUpgradeReport(path string, report engine.ApplyReport) {
	r.renderApplyReport(path, report)
}
