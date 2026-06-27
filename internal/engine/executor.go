package engine

import "context"

// ApplySummary aggregates apply results.
type ApplySummary struct {
	Total     int `json:"total"`
	Changed   int `json:"changed"`
	Noop      int `json:"noop"`
	Skipped   int `json:"skipped"`
	Failed    int `json:"failed"`
	Applied   int `json:"applied"`
	Unchanged int `json:"unchanged"`
}

// ApplyItem captures one resource apply outcome.
type ApplyItem struct {
	ResourceID string
	Type       string
	Action     string
	Changed    bool
	Message    string
	Error      string
	Details    map[string]string
}

// ApplyReport is the result of executing a plan.
type ApplyReport struct {
	Items   []ApplyItem
	Summary ApplySummary
}

// ApplyObserver receives progress events while the executor applies resources.
type ApplyObserver interface {
	BeforeApply(item PlanItem)
}

// HasFailures reports whether any apply action failed.
func (report ApplyReport) HasFailures() bool {
	return report.Summary.Failed > 0
}

// Executor applies resources selected by a plan.
type Executor struct{}

// NewExecutor returns a sequential executor.
func NewExecutor() Executor {
	return Executor{}
}

// Apply runs Apply for resources whose plan action is ActionApply.
func (executor Executor) Apply(ctx context.Context, resources []Resource, plan Plan) ApplyReport {
	return executor.ApplyWithObserver(ctx, resources, plan, nil)
}

// ApplyWithObserver runs Apply and reports progress before each mutating action.
func (executor Executor) ApplyWithObserver(ctx context.Context, resources []Resource, plan Plan, observer ApplyObserver) ApplyReport {
	if items := duplicateResourceIDApplyItems(resources); len(items) > 0 {
		report := ApplyReport{
			Items: make([]ApplyItem, 0, len(items)),
		}
		for _, item := range items {
			report.Items = append(report.Items, item)
			report.Summary.add(item)
		}
		return report
	}

	resourceByID := make(map[string]Resource, len(resources))
	for _, resource := range resources {
		resourceByID[resource.ID()] = resource
	}

	report := ApplyReport{
		Items: make([]ApplyItem, 0, len(plan.Items)),
	}

	for i, planItem := range plan.Items {
		resource := resourceByID[planItem.ResourceID]
		if planItem.Action != ActionApply {
			item := applyItemFromPlan(planItem)
			report.Items = append(report.Items, item)
			report.Summary.add(item)
			if planItem.Action == ActionFail && resourceBlocksApply(resource) {
				appendBlockedApplyItems(&report, plan.Items[i+1:], planItem.ResourceID)
				return report
			}
			continue
		}

		if resource == nil {
			item := ApplyItem{
				ResourceID: planItem.ResourceID,
				Type:       planItem.Type,
				Action:     "fail",
				Message:    "planned resource was not found",
				Error:      "planned resource was not found",
				Details:    planItem.Details,
			}
			report.Items = append(report.Items, item)
			report.Summary.add(item)
			continue
		}

		if observer != nil {
			observer.BeforeApply(planItem)
		}

		result, err := resource.Apply(ctx)
		item := applyItemFor(resource, result, err)
		report.Items = append(report.Items, item)
		report.Summary.add(item)
		if err != nil && resourceBlocksApply(resource) {
			appendBlockedApplyItems(&report, plan.Items[i+1:], resource.ID())
			return report
		}
	}

	return report
}

func appendBlockedApplyItems(report *ApplyReport, planItems []PlanItem, blockerID string) {
	for _, planItem := range planItems {
		item := applyItemFromPlan(planItem)
		if planItem.Action == ActionApply {
			item = ApplyItem{
				ResourceID: planItem.ResourceID,
				Type:       planItem.Type,
				Action:     "skip",
				Message:    "blocked by " + blockerID,
				Details:    planItem.Details,
			}
		}
		report.Items = append(report.Items, item)
		report.Summary.add(item)
	}
}

func resourceBlocksApply(resource Resource) bool {
	if resource == nil {
		return false
	}
	blocker, ok := resource.(ApplyBlocker)
	return ok && blocker.BlocksApply()
}

func applyItemFromPlan(planItem PlanItem) ApplyItem {
	action := "noop"
	switch planItem.Action {
	case ActionSkip:
		action = "skip"
	case ActionFail:
		action = "fail"
	}

	return ApplyItem{
		ResourceID: planItem.ResourceID,
		Type:       planItem.Type,
		Action:     action,
		Changed:    false,
		Message:    planItem.Message,
		Error:      planItem.Error,
		Details:    planItem.Details,
	}
}

func applyItemFor(resource Resource, result ApplyResult, err error) ApplyItem {
	if result.ResourceID == "" {
		result.ResourceID = resource.ID()
	}
	if result.Type == "" {
		result.Type = resource.Type()
	}
	if result.Action == "" {
		result.Action = "unknown"
	}

	item := ApplyItem{
		ResourceID: result.ResourceID,
		Type:       result.Type,
		Action:     result.Action,
		Changed:    result.Changed,
		Message:    result.Message,
		Details:    result.Details,
	}

	if err != nil {
		item.Error = err.Error()
		if item.Action == "" || item.Action == "unknown" {
			item.Action = "fail"
		}
		if item.Message == "" {
			item.Message = item.Error
		}
	}

	return item
}

func (summary *ApplySummary) add(item ApplyItem) {
	summary.Total++

	switch item.Action {
	case "skip":
		summary.Skipped++
	case "noop":
		summary.Noop++
	case "fail":
		summary.Failed++
	default:
		if item.Error != "" {
			summary.Failed++
		} else if item.Changed {
			summary.Changed++
		} else {
			summary.Noop++
		}
	}

	if item.Changed {
		summary.Applied++
	} else {
		summary.Unchanged++
	}
}
