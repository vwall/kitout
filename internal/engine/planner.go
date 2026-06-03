package engine

import "context"

// PlanAction describes the action the engine should take for a status result.
type PlanAction string

const (
	ActionNoop  PlanAction = "noop"
	ActionApply PlanAction = "apply"
	ActionFail  PlanAction = "fail"
	ActionSkip  PlanAction = "skip"
)

// PlanItem describes the planned action for one resource.
type PlanItem struct {
	ResourceID string
	Type       string
	State      ResourceState
	Action     PlanAction
	Message    string
	Error      string
	Details    map[string]string
}

// PlanSummary aggregates resource states and planned actions.
type PlanSummary struct {
	Total     int `json:"total"`
	Satisfied int `json:"satisfied"`
	Missing   int `json:"missing"`
	Changed   int `json:"changed"`
	Failed    int `json:"failed"`
	Skipped   int `json:"skipped"`
	Unknown   int `json:"unknown"`
	ToApply   int `json:"to_apply"`
}

// Plan is the result of checking resources and mapping their states to actions.
type Plan struct {
	Items   []PlanItem
	Summary PlanSummary
}

// HasChanges reports whether the plan contains resources that should be applied.
func (p Plan) HasChanges() bool {
	return p.Summary.ToApply > 0
}

// HasFailures reports whether the plan contains resources that cannot be applied.
func (p Plan) HasFailures() bool {
	return p.Summary.Failed > 0 || p.Summary.Unknown > 0
}

// Planner checks resources and builds a mutation-free plan.
type Planner struct{}

// NewPlanner returns a planner with default status-to-action rules.
func NewPlanner() Planner {
	return Planner{}
}

// Build checks resource status and builds a plan without applying changes.
func (p Planner) Build(ctx context.Context, resources []Resource) Plan {
	plan := Plan{
		Items: make([]PlanItem, 0, len(resources)),
	}

	for _, resource := range resources {
		result, err := resource.Status(ctx)
		item := planItemFor(resource, result, err)
		plan.Items = append(plan.Items, item)
		plan.Summary.add(item)
	}

	return plan
}

func planItemFor(resource Resource, result StatusResult, err error) PlanItem {
	if result.ResourceID == "" {
		result.ResourceID = resource.ID()
	}
	if result.Type == "" {
		result.Type = resource.Type()
	}
	if result.State == "" {
		result.State = StateUnknown
	}

	item := PlanItem{
		ResourceID: result.ResourceID,
		Type:       result.Type,
		State:      result.State,
		Action:     actionForState(result.State),
		Message:    result.Message,
		Details:    result.Details,
	}

	if err != nil {
		item.State = StateFailed
		item.Action = ActionFail
		item.Error = err.Error()
		if item.Message == "" {
			item.Message = item.Error
		}
	}

	return item
}

func actionForState(state ResourceState) PlanAction {
	switch state {
	case StateSatisfied:
		return ActionNoop
	case StateMissing, StateChanged:
		return ActionApply
	case StateSkipped:
		return ActionSkip
	case StateFailed, StateUnknown:
		return ActionFail
	default:
		return ActionFail
	}
}

func (summary *PlanSummary) add(item PlanItem) {
	summary.Total++

	switch item.State {
	case StateSatisfied:
		summary.Satisfied++
	case StateMissing:
		summary.Missing++
	case StateChanged:
		summary.Changed++
	case StateFailed:
		summary.Failed++
	case StateSkipped:
		summary.Skipped++
	case StateUnknown:
		summary.Unknown++
	default:
		summary.Unknown++
	}

	if item.Action == ActionApply {
		summary.ToApply++
	}
}
