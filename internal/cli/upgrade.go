package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

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

func (app application) runUpgrade(args []string, opts globalOptions) int {
	stdout, stderr := app.stdout, app.stderr
	var upgradeOpts upgradeOptions
	fs := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	fs.SetOutput(stderr)
	addGlobalFlags(fs, &opts)
	fs.BoolVar(&upgradeOpts.dryRun, "dry-run", upgradeOpts.dryRun, "Show planned upgrades without applying them")
	fs.StringVar(&upgradeOpts.only, "only", upgradeOpts.only, "Limit upgrades to brew or cask")

	if err := fs.Parse(args); err != nil {
		return exitValidation
	}

	targets := fs.Args()
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

	planResources, err := filterUpgradeResources(resources.Build(loaded.Config, app.newRunner()), only, targets)
	if err != nil {
		return renderUpgradeValidationError(err, opts, jsonRenderer, stderr)
	}

	if !opts.json {
		renderer.renderUpgradePlanStart(loaded.Path, upgradeOpts.dryRun)
		renderer.renderConfigWarnings(loaded.Warnings)
	}

	var planObserver engine.PlanObserver
	if !opts.json {
		planObserver = newApplyPlanObserver(renderer, opts.verbose)
	}
	plan := buildUpgradePlan(app.ctx, planResources, planObserver)

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
	upgradeRunner := app.newRunner()
	if verboseUpgradeOutputEnabled(opts, upgradeOpts) && plan.Summary.ToApply > 0 {
		upgradeRunner = app.newVerboseRunner()
	}
	rawUpgradeResources, err := filterUpgradeResources(resources.BuildUncached(loaded.Config, upgradeRunner), only, targets)
	if err != nil {
		return renderUpgradeValidationError(err, opts, jsonRenderer, stderr)
	}
	upgradeResources := wrapUpgradeResources(rawUpgradeResources)
	report := engine.NewExecutor().ApplyWithObserver(app.ctx, upgradeResources, plan, observer)
	if opts.json {
		if err := jsonRenderer.renderUpgradeReport(loaded.Path, loaded.Warnings, report); err != nil {
			fmt.Fprintf(stderr, "Failed to render JSON: %v\n", err)
			return exitRuntimeError
		}
		return executionReportExitCode(report)
	}

	renderer.renderUpgradeReport(reportPath, report)
	return executionReportExitCode(report)
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

func filterUpgradeResources(resourceList []engine.Resource, only string, targets []string) ([]engine.Resource, error) {
	targetSet := stringSet(targets)
	if len(targetSet) > 0 {
		if err := validateUpgradeTargets(resourceList, only, uniqueStrings(targets)); err != nil {
			return nil, err
		}
	}

	filtered := make([]engine.Resource, 0, len(resourceList))
	for _, resource := range resourceList {
		if resource.Type() != "brew" && resource.Type() != "cask" {
			continue
		}
		if only != "" && resource.Type() != only {
			continue
		}
		if len(targetSet) > 0 {
			if _, ok := targetSet[resource.ID()]; !ok {
				continue
			}
		}
		filtered = append(filtered, resource)
	}
	return filtered, nil
}

func validateUpgradeTargets(resourceList []engine.Resource, only string, targets []string) error {
	typesByID := make(map[string]string, len(resourceList))
	for _, resource := range resourceList {
		if _, exists := typesByID[resource.ID()]; !exists {
			typesByID[resource.ID()] = resource.Type()
		}
	}

	var unknown []string
	var unsupported []string
	var excluded []string
	for _, target := range targets {
		typ, ok := typesByID[target]
		if !ok {
			unknown = append(unknown, target)
			continue
		}
		if typ != "brew" && typ != "cask" {
			unsupported = append(unsupported, target)
			continue
		}
		if only != "" && typ != only {
			excluded = append(excluded, target)
		}
	}

	if len(unknown) > 0 {
		return fmt.Errorf("unknown upgrade %s %s; use configured brew:<name> or cask:<name> resource IDs", pluralize("target", len(unknown)), quoteList(unknown))
	}
	if len(unsupported) > 0 {
		return fmt.Errorf("upgrade %s %s %s not support upgrade; use configured brew:<name> or cask:<name> resource IDs", pluralize("target", len(unsupported)), quoteList(unsupported), pluralVerb("does", "do", len(unsupported)))
	}
	if len(excluded) > 0 {
		return fmt.Errorf("upgrade %s %s %s excluded by --only %s", pluralize("target", len(excluded)), quoteList(excluded), pluralVerb("is", "are", len(excluded)), only)
	}
	return nil
}

func stringSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	unique := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	return unique
}

func quoteList(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, strconv.Quote(value))
	}
	return strings.Join(quoted, ", ")
}

func pluralize(word string, count int) string {
	if count == 1 {
		return word
	}
	return word + "s"
}

func pluralVerb(singular, plural string, count int) string {
	if count == 1 {
		return singular
	}
	return plural
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
	plan := engine.NewPlanFromItems(items)
	plan.ExecutionError = statusPlan.ExecutionError
	return plan
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
	if plan.ExecutionError != "" {
		fmt.Fprintf(r.stdout, "%s Execution stopped: %s\n", r.failSymbol(), plan.ExecutionError)
		return
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
