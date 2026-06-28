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

type agentExplainReport struct {
	ConfigPath       string
	Resource         agentResourceSummary
	Item             engine.PlanItem
	RiskyApply       bool
	ApprovalReason   string
	RelatedCommands  []agentCommand
	AgentGuidance    []string
	ConfigWarnings   []config.ConfigWarning
	ResourceWasFound bool
}

func runExplain(args []string, opts globalOptions, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("explain", flag.ContinueOnError)
	fs.SetOutput(stderr)
	addGlobalFlags(fs, &opts)

	if err := fs.Parse(args); err != nil {
		return exitValidation
	}
	if fs.NArg() != 1 {
		if opts.json {
			if err := newJSONRenderer(stdout).renderValidationMessage("explain", "usage: kitout explain <resource-id>"); err != nil {
				fmt.Fprintf(stderr, "Failed to render JSON: %v\n", err)
				return exitRuntimeError
			}
		} else {
			fmt.Fprintln(stderr, "usage: kitout explain <resource-id>")
		}
		return exitValidation
	}
	resourceID := fs.Arg(0)

	renderer := newHumanRenderer(stdout, stderr, opts)
	jsonRenderer := newJSONRenderer(stdout)
	configPath, err := config.SelectPath(opts.configPath)
	if err != nil {
		return renderConfigError("explain", err, opts, renderer, jsonRenderer, stderr)
	}
	loaded, err := config.LoadFile(configPath)
	if err != nil {
		return renderConfigError("explain", err, opts, renderer, jsonRenderer, stderr)
	}

	resourceList := resources.Build(loaded.Config, newCLIExecRunner())
	resource, ok := findResourceByID(resourceList, resourceID)
	if !ok {
		if opts.json {
			if err := jsonRenderer.renderValidationMessage("explain", fmt.Sprintf("resource %q is not configured", resourceID)); err != nil {
				fmt.Fprintf(stderr, "Failed to render JSON: %v\n", err)
				return exitRuntimeError
			}
		} else {
			fmt.Fprintf(stderr, "resource %q is not configured\n", resourceID)
		}
		return exitValidation
	}

	var observer engine.PlanObserver
	if !opts.json && opts.verbose {
		observer = renderer
	}
	plan := engine.NewPlanner().BuildWithObserver(context.Background(), []engine.Resource{resource}, observer)
	report := buildAgentExplainReport(loaded, resourceID, plan)
	if opts.json {
		if err := jsonRenderer.renderAgentExplain(report); err != nil {
			fmt.Fprintf(stderr, "Failed to render JSON: %v\n", err)
			return exitRuntimeError
		}
		if plan.HasFailures() {
			return exitRuntimeError
		}
		return exitOK
	}

	renderer.renderConfigWarnings(loaded.Warnings)
	renderer.renderAgentExplain(report)
	if plan.HasFailures() {
		return exitRuntimeError
	}
	return exitOK
}

func findResourceByID(resources []engine.Resource, id string) (engine.Resource, bool) {
	for _, resource := range resources {
		if resource.ID() == id {
			return resource, true
		}
	}
	return nil, false
}

func buildAgentExplainReport(loaded config.LoadedConfig, resourceID string, plan engine.Plan) agentExplainReport {
	item := engine.PlanItem{ResourceID: resourceID}
	if len(plan.Items) > 0 {
		item = plan.Items[0]
	}
	riskyItems := riskyApplyItems(engine.Plan{Items: []engine.PlanItem{item}})
	risky := len(riskyItems) > 0
	approvalReason := ""
	if risky {
		approvalReason = riskyApplyReason(item)
	}

	return agentExplainReport{
		ConfigPath: loaded.Path,
		Resource: agentResourceSummary{
			ResourceID: item.ResourceID,
			Type:       item.Type,
			Label:      displayResourceLabel(item.Type, item.ResourceID, item.Details),
			Details:    item.Details,
		},
		Item:             item,
		RiskyApply:       risky,
		ApprovalReason:   approvalReason,
		RelatedCommands:  explainRelatedCommands(loaded.Path, item.ResourceID),
		AgentGuidance:    explainAgentGuidance(item),
		ConfigWarnings:   loaded.Warnings,
		ResourceWasFound: len(plan.Items) > 0,
	}
}

func riskyApplyReason(item engine.PlanItem) string {
	switch item.Type {
	case "shell":
		return "shell resources run explicit configured commands during apply"
	case "login_shell":
		return "login shell changes affect the current user's account"
	case "security":
		return "security resources change macOS security settings"
	case "system":
		return "system resources may start macOS installers or privileged system changes"
	case "ssh_key":
		return "SSH key resources create key material on disk"
	case "copy":
		return "copy replacement may overwrite an existing target"
	case "symlink":
		return "symlink replacement may remove an existing target"
	default:
		return "this resource is treated as risky during apply"
	}
}

func explainRelatedCommands(configPath, resourceID string) []agentCommand {
	return []agentCommand{
		{
			Command: "kitout status --config " + quoteCommandArg(configPath) + " --json",
			Reason:  "Inspect all configured resources as JSON.",
		},
		{
			Command: "kitout apply --config " + quoteCommandArg(configPath) + " --dry-run --json",
			Reason:  "Preview all changes without applying them.",
		},
		{
			Command: "kitout explain --config " + quoteCommandArg(configPath) + " --json " + quoteCommandArg(resourceID),
			Reason:  "Explain this resource in stable JSON.",
		},
	}
}

func explainAgentGuidance(item engine.PlanItem) []string {
	guidance := []string{
		"Use status or dry-run output as evidence when answering the user.",
		"Ask the user before running kitout apply.",
	}
	if item.Type == "symlink" || item.Type == "copy" {
		guidance = append(guidance, "When changing managed dotfiles, edit the source path instead of the target path.")
	}
	if item.Type == "shell" {
		guidance = append(guidance, "Explain the configured shell command before any apply run.")
	}
	return guidance
}
