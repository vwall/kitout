package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/vwall/kitout/internal/config"
	"github.com/vwall/kitout/internal/engine"
)

type jsonRenderer struct {
	stdout io.Writer
}

type jsonStatusResponse struct {
	Command string            `json:"command"`
	OK      bool              `json:"ok"`
	Config  *jsonConfigStatus `json:"config,omitempty"`
	Plan    *jsonPlan         `json:"plan,omitempty"`
	Apply   *jsonApplyReport  `json:"apply,omitempty"`
	Upgrade *jsonApplyReport  `json:"upgrade,omitempty"`
	Doctor  *jsonDoctorReport `json:"doctor,omitempty"`
	Context *jsonAgentContext `json:"context,omitempty"`
	Explain *jsonAgentExplain `json:"explain,omitempty"`
	Error   *jsonError        `json:"error,omitempty"`
}

type jsonConfigStatus struct {
	Path     string            `json:"path,omitempty"`
	Valid    bool              `json:"valid"`
	Warnings []jsonErrorDetail `json:"warnings,omitempty"`
}

type jsonPlan struct {
	Summary engine.PlanSummary `json:"summary"`
	Items   []jsonPlanItem     `json:"items"`
	DryRun  bool               `json:"dry_run,omitempty"`
}

type jsonPlanItem struct {
	ResourceID string            `json:"resource_id"`
	Type       string            `json:"type"`
	State      string            `json:"state"`
	Action     string            `json:"action"`
	Message    string            `json:"message,omitempty"`
	Error      string            `json:"error,omitempty"`
	Details    map[string]string `json:"details,omitempty"`
	Advisories []jsonAdvisory    `json:"advisories,omitempty"`
}

type jsonApplyReport struct {
	Summary engine.ApplySummary `json:"summary"`
	Items   []jsonApplyItem     `json:"items"`
}

type jsonApplyItem struct {
	ResourceID string            `json:"resource_id"`
	Type       string            `json:"type"`
	Action     string            `json:"action"`
	Changed    bool              `json:"changed"`
	Message    string            `json:"message,omitempty"`
	Error      string            `json:"error,omitempty"`
	Details    map[string]string `json:"details,omitempty"`
}

type jsonDoctorReport struct {
	Summary doctorSummary    `json:"summary"`
	Items   []jsonDoctorItem `json:"items"`
}

type jsonAgentContext struct {
	SchemaVersion    int                 `json:"schema_version"`
	ConfigDir        string              `json:"config_dir"`
	SafeCommands     []jsonAgentCommand  `json:"safe_commands"`
	RequiresApproval []jsonAgentCommand  `json:"requires_approval"`
	Resources        []jsonAgentResource `json:"resources"`
	Guidance         []string            `json:"guidance"`
}

type jsonAgentExplain struct {
	Resource        jsonAgentResource  `json:"resource"`
	Status          jsonPlanItem       `json:"status"`
	Safety          jsonExplainSafety  `json:"safety"`
	RelatedCommands []jsonAgentCommand `json:"related_commands"`
	Guidance        []string           `json:"guidance"`
}

type jsonAgentCommand struct {
	Command string `json:"command"`
	Reason  string `json:"reason"`
}

type jsonAgentResource struct {
	ResourceID string            `json:"resource_id"`
	Type       string            `json:"type"`
	Label      string            `json:"label"`
	Details    map[string]string `json:"details,omitempty"`
}

type jsonExplainSafety struct {
	WouldApply       bool   `json:"would_apply"`
	RequiresApproval bool   `json:"requires_approval"`
	Reason           string `json:"reason,omitempty"`
}

type jsonDoctorItem struct {
	Name    string            `json:"name"`
	State   string            `json:"state"`
	Message string            `json:"message"`
	Fix     string            `json:"fix,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

type jsonError struct {
	Type    string            `json:"type"`
	Message string            `json:"message"`
	Details []jsonErrorDetail `json:"details,omitempty"`
}

type jsonErrorDetail struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

type jsonAdvisory struct {
	Code     string            `json:"code"`
	Severity string            `json:"severity"`
	Message  string            `json:"message"`
	Fix      string            `json:"fix,omitempty"`
	Details  map[string]string `json:"details,omitempty"`
}

func newJSONRenderer(stdout io.Writer) jsonRenderer {
	return jsonRenderer{stdout: stdout}
}

func (r jsonRenderer) renderPlan(command, path string, warnings []config.ConfigWarning, plan engine.Plan, dryRun bool) error {
	return r.write(jsonStatusResponse{
		Command: command,
		OK:      !plan.HasFailures(),
		Config: &jsonConfigStatus{
			Path:     path,
			Valid:    true,
			Warnings: jsonWarningsFromConfig(warnings),
		},
		Plan: jsonPlanFromEngine(plan, dryRun),
	})
}

func (r jsonRenderer) renderApplyReport(path string, warnings []config.ConfigWarning, report engine.ApplyReport) error {
	return r.write(jsonStatusResponse{
		Command: "apply",
		OK:      !report.HasFailures(),
		Config: &jsonConfigStatus{
			Path:     path,
			Valid:    true,
			Warnings: jsonWarningsFromConfig(warnings),
		},
		Apply: jsonApplyReportFromEngine(report),
	})
}

func (r jsonRenderer) renderUpgradeReport(path string, warnings []config.ConfigWarning, report engine.ApplyReport) error {
	return r.write(jsonStatusResponse{
		Command: "upgrade",
		OK:      !report.HasFailures(),
		Config: &jsonConfigStatus{
			Path:     path,
			Valid:    true,
			Warnings: jsonWarningsFromConfig(warnings),
		},
		Upgrade: jsonApplyReportFromEngine(report),
	})
}

func (r jsonRenderer) renderDoctorReport(report doctorReport) error {
	return r.write(jsonStatusResponse{
		Command: "doctor",
		OK:      !report.HasFailures(),
		Config: &jsonConfigStatus{
			Path:     report.ConfigPath,
			Valid:    report.configIsValid(),
			Warnings: jsonWarningsFromConfig(report.ConfigWarnings),
		},
		Doctor: jsonDoctorReportFromCLI(report),
	})
}

func (r jsonRenderer) renderAgentContext(report agentContextReport) error {
	return r.write(jsonStatusResponse{
		Command: "context",
		OK:      true,
		Config: &jsonConfigStatus{
			Path:     report.ConfigPath,
			Valid:    true,
			Warnings: jsonWarningsFromConfig(report.ConfigWarnings),
		},
		Context: jsonAgentContextFromCLI(report),
	})
}

func (r jsonRenderer) renderAgentExplain(report agentExplainReport) error {
	return r.write(jsonStatusResponse{
		Command: "explain",
		OK:      report.ResourceWasFound && report.Item.State != engine.StateFailed && report.Item.State != engine.StateUnknown,
		Config: &jsonConfigStatus{
			Path:     report.ConfigPath,
			Valid:    true,
			Warnings: jsonWarningsFromConfig(report.ConfigWarnings),
		},
		Explain: jsonAgentExplainFromCLI(report),
	})
}

func (r jsonRenderer) renderValidationMessage(command, message string) error {
	return r.write(jsonStatusResponse{
		Command: command,
		OK:      false,
		Error: &jsonError{
			Type:    "validation",
			Message: message,
		},
	})
}

func (r jsonRenderer) renderValidationErrors(command string, errs config.ValidationErrors) error {
	details := make([]jsonErrorDetail, 0, len(errs))
	for _, err := range errs {
		details = append(details, jsonErrorDetail{
			Field:   err.Field,
			Message: err.Message,
		})
	}

	return r.write(jsonStatusResponse{
		Command: command,
		OK:      false,
		Config: &jsonConfigStatus{
			Valid: false,
		},
		Error: &jsonError{
			Type:    "validation",
			Message: errs.Error(),
			Details: details,
		},
	})
}

func (r jsonRenderer) renderParseError(command string, err config.ParseError) error {
	return r.write(jsonStatusResponse{
		Command: command,
		OK:      false,
		Config: &jsonConfigStatus{
			Path:  err.Path,
			Valid: false,
		},
		Error: &jsonError{
			Type:    "parse",
			Message: fmt.Sprintf("Invalid config: %v", err),
		},
	})
}

func (r jsonRenderer) renderConfigLoadFailure(command string, err error) error {
	return r.write(jsonStatusResponse{
		Command: command,
		OK:      false,
		Error: &jsonError{
			Type:    "runtime",
			Message: fmt.Sprintf("Failed to load config: %v", err),
		},
	})
}

func (r jsonRenderer) write(response jsonStatusResponse) error {
	encoder := json.NewEncoder(r.stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(response)
}

func jsonWarningsFromConfig(warnings []config.ConfigWarning) []jsonErrorDetail {
	if len(warnings) == 0 {
		return nil
	}

	details := make([]jsonErrorDetail, 0, len(warnings))
	for _, warning := range warnings {
		details = append(details, jsonErrorDetail{
			Field:   warning.Field,
			Message: warning.Message,
		})
	}
	return details
}

func jsonPlanFromEngine(plan engine.Plan, dryRun bool) *jsonPlan {
	items := make([]jsonPlanItem, 0, len(plan.Items))
	for _, item := range plan.Items {
		items = append(items, jsonPlanItem{
			ResourceID: item.ResourceID,
			Type:       item.Type,
			State:      string(item.State),
			Action:     string(item.Action),
			Message:    item.Message,
			Error:      item.Error,
			Details:    item.Details,
			Advisories: jsonAdvisoriesFromEngine(item.Advisories),
		})
	}

	return &jsonPlan{
		Summary: plan.Summary,
		Items:   items,
		DryRun:  dryRun,
	}
}

func jsonAdvisoriesFromEngine(advisories []engine.Advisory) []jsonAdvisory {
	if len(advisories) == 0 {
		return nil
	}

	items := make([]jsonAdvisory, 0, len(advisories))
	for _, advisory := range advisories {
		items = append(items, jsonAdvisory{
			Code:     advisory.Code,
			Severity: string(advisory.Severity),
			Message:  advisory.Message,
			Fix:      advisory.Fix,
			Details:  advisory.Details,
		})
	}
	return items
}

func jsonApplyReportFromEngine(report engine.ApplyReport) *jsonApplyReport {
	items := make([]jsonApplyItem, 0, len(report.Items))
	for _, item := range report.Items {
		items = append(items, jsonApplyItem{
			ResourceID: item.ResourceID,
			Type:       item.Type,
			Action:     item.Action,
			Changed:    item.Changed,
			Message:    item.Message,
			Error:      item.Error,
			Details:    item.Details,
		})
	}

	return &jsonApplyReport{
		Summary: report.Summary,
		Items:   items,
	}
}

func jsonDoctorReportFromCLI(report doctorReport) *jsonDoctorReport {
	items := make([]jsonDoctorItem, 0, len(report.Items))
	for _, item := range report.Items {
		items = append(items, jsonDoctorItem{
			Name:    item.Name,
			State:   string(item.State),
			Message: item.Message,
			Fix:     item.Fix,
			Details: item.Details,
		})
	}

	return &jsonDoctorReport{
		Summary: report.Summary,
		Items:   items,
	}
}

func jsonAgentContextFromCLI(report agentContextReport) *jsonAgentContext {
	return &jsonAgentContext{
		SchemaVersion:    report.SchemaVersion,
		ConfigDir:        report.ConfigDir,
		SafeCommands:     jsonAgentCommandsFromCLI(report.SafeCommands),
		RequiresApproval: jsonAgentCommandsFromCLI(report.RequiresApproval),
		Resources:        jsonAgentResourcesFromCLI(report.ManagedResources),
		Guidance:         report.Guidance,
	}
}

func jsonAgentExplainFromCLI(report agentExplainReport) *jsonAgentExplain {
	return &jsonAgentExplain{
		Resource: jsonAgentResourceFromCLI(report.Resource),
		Status: jsonPlanItem{
			ResourceID: report.Item.ResourceID,
			Type:       report.Item.Type,
			State:      string(report.Item.State),
			Action:     string(report.Item.Action),
			Message:    report.Item.Message,
			Error:      report.Item.Error,
			Details:    report.Item.Details,
			Advisories: jsonAdvisoriesFromEngine(report.Item.Advisories),
		},
		Safety: jsonExplainSafety{
			WouldApply:       report.Item.Action == engine.ActionApply,
			RequiresApproval: report.RiskyApply,
			Reason:           report.ApprovalReason,
		},
		RelatedCommands: jsonAgentCommandsFromCLI(report.RelatedCommands),
		Guidance:        report.AgentGuidance,
	}
}

func jsonAgentCommandsFromCLI(commands []agentCommand) []jsonAgentCommand {
	items := make([]jsonAgentCommand, 0, len(commands))
	for _, command := range commands {
		items = append(items, jsonAgentCommand{
			Command: command.Command,
			Reason:  command.Reason,
		})
	}
	return items
}

func jsonAgentResourcesFromCLI(resources []agentResourceSummary) []jsonAgentResource {
	items := make([]jsonAgentResource, 0, len(resources))
	for _, resource := range resources {
		items = append(items, jsonAgentResourceFromCLI(resource))
	}
	return items
}

func jsonAgentResourceFromCLI(resource agentResourceSummary) jsonAgentResource {
	return jsonAgentResource{
		ResourceID: resource.ResourceID,
		Type:       resource.Type,
		Label:      resource.Label,
		Details:    resource.Details,
	}
}

func (report doctorReport) configIsValid() bool {
	for _, item := range report.Items {
		if item.Name == "Config" {
			return item.State != doctorFail
		}
	}
	return false
}
