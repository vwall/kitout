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
	Error   *jsonError        `json:"error,omitempty"`
}

type jsonConfigStatus struct {
	Path  string `json:"path,omitempty"`
	Valid bool   `json:"valid"`
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

type jsonError struct {
	Type    string            `json:"type"`
	Message string            `json:"message"`
	Details []jsonErrorDetail `json:"details,omitempty"`
}

type jsonErrorDetail struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

func newJSONRenderer(stdout io.Writer) jsonRenderer {
	return jsonRenderer{stdout: stdout}
}

func (r jsonRenderer) renderPlan(command, path string, plan engine.Plan, dryRun bool) error {
	return r.write(jsonStatusResponse{
		Command: command,
		OK:      !plan.HasFailures(),
		Config: &jsonConfigStatus{
			Path:  path,
			Valid: true,
		},
		Plan: jsonPlanFromEngine(plan, dryRun),
	})
}

func (r jsonRenderer) renderApplyReport(path string, report engine.ApplyReport) error {
	return r.write(jsonStatusResponse{
		Command: "apply",
		OK:      !report.HasFailures(),
		Config: &jsonConfigStatus{
			Path:  path,
			Valid: true,
		},
		Apply: jsonApplyReportFromEngine(report),
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
		})
	}

	return &jsonPlan{
		Summary: plan.Summary,
		Items:   items,
		DryRun:  dryRun,
	}
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
