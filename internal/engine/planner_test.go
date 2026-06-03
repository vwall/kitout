package engine

import (
	"context"
	"errors"
	"testing"
)

func TestPlannerBuildsPlanFromStatusResults(t *testing.T) {
	resources := []Resource{
		&fakeResource{id: "directory:/Users/nix/code", typ: "directory", state: StateSatisfied, message: "exists"},
		&fakeResource{id: "brew:git", typ: "brew", state: StateMissing, message: "not installed"},
		&fakeResource{id: "symlink:/Users/nix/.zshrc", typ: "symlink", state: StateChanged, message: "points elsewhere"},
		&fakeResource{id: "shell:corepack", typ: "shell", state: StateSkipped, message: "condition not met"},
	}

	plan := NewPlanner().Build(context.Background(), resources)

	if got, want := len(plan.Items), 4; got != want {
		t.Fatalf("len(plan.Items) = %d, want %d", got, want)
	}
	assertPlanItem(t, plan.Items[0], "directory:/Users/nix/code", "directory", StateSatisfied, ActionNoop)
	assertPlanItem(t, plan.Items[1], "brew:git", "brew", StateMissing, ActionApply)
	assertPlanItem(t, plan.Items[2], "symlink:/Users/nix/.zshrc", "symlink", StateChanged, ActionApply)
	assertPlanItem(t, plan.Items[3], "shell:corepack", "shell", StateSkipped, ActionSkip)

	wantSummary := PlanSummary{
		Total:     4,
		Satisfied: 1,
		Missing:   1,
		Changed:   1,
		Skipped:   1,
		ToApply:   2,
	}
	if plan.Summary != wantSummary {
		t.Fatalf("summary = %+v, want %+v", plan.Summary, wantSummary)
	}
	if !plan.HasChanges() {
		t.Fatalf("HasChanges() = false, want true")
	}
	if plan.HasFailures() {
		t.Fatalf("HasFailures() = true, want false")
	}
}

func TestPlannerTreatsStatusErrorsAsFailedPlanItems(t *testing.T) {
	resources := []Resource{
		&fakeResource{id: "brew:git", typ: "brew", state: StateMissing},
		&fakeResource{id: "repo:/Users/nix/code/aubs", typ: "repo", err: errors.New("git not found")},
	}

	plan := NewPlanner().Build(context.Background(), resources)

	if got, want := len(plan.Items), 2; got != want {
		t.Fatalf("len(plan.Items) = %d, want %d", got, want)
	}
	assertPlanItem(t, plan.Items[1], "repo:/Users/nix/code/aubs", "repo", StateFailed, ActionFail)
	if plan.Items[1].Error != "git not found" {
		t.Fatalf("error = %q, want git not found", plan.Items[1].Error)
	}

	wantSummary := PlanSummary{
		Total:   2,
		Missing: 1,
		Failed:  1,
		ToApply: 1,
	}
	if plan.Summary != wantSummary {
		t.Fatalf("summary = %+v, want %+v", plan.Summary, wantSummary)
	}
	if !plan.HasFailures() {
		t.Fatalf("HasFailures() = false, want true")
	}
}

func TestPlannerNeverAppliesResources(t *testing.T) {
	resource := &fakeResource{
		id:    "directory:/Users/nix/code",
		typ:   "directory",
		state: StateMissing,
	}

	plan := NewPlanner().Build(context.Background(), []Resource{resource})

	if resource.statusCalls != 1 {
		t.Fatalf("statusCalls = %d, want 1", resource.statusCalls)
	}
	if resource.applyCalls != 0 {
		t.Fatalf("applyCalls = %d, want 0", resource.applyCalls)
	}
	if !plan.HasChanges() {
		t.Fatalf("HasChanges() = false, want true")
	}
}

func TestPlannerDefaultsIncompleteStatusResults(t *testing.T) {
	resource := &fakeResource{id: "mystery:thing", typ: "mystery"}

	plan := NewPlanner().Build(context.Background(), []Resource{resource})

	assertPlanItem(t, plan.Items[0], "mystery:thing", "mystery", StateUnknown, ActionFail)
	if !plan.HasFailures() {
		t.Fatalf("HasFailures() = false, want true")
	}
	if plan.Summary.Unknown != 1 {
		t.Fatalf("Unknown = %d, want 1", plan.Summary.Unknown)
	}
}

func assertPlanItem(t *testing.T, item PlanItem, id, typ string, state ResourceState, action PlanAction) {
	t.Helper()

	if item.ResourceID != id {
		t.Fatalf("ResourceID = %q, want %q", item.ResourceID, id)
	}
	if item.Type != typ {
		t.Fatalf("Type = %q, want %q", item.Type, typ)
	}
	if item.State != state {
		t.Fatalf("State = %q, want %q", item.State, state)
	}
	if item.Action != action {
		t.Fatalf("Action = %q, want %q", item.Action, action)
	}
}

type fakeResource struct {
	id          string
	typ         string
	state       ResourceState
	message     string
	err         error
	applyErr    error
	statusCalls int
	applyCalls  int
}

func (resource *fakeResource) ID() string {
	return resource.id
}

func (resource *fakeResource) Type() string {
	return resource.typ
}

func (resource *fakeResource) Status(ctx context.Context) (StatusResult, error) {
	resource.statusCalls++
	return StatusResult{
		ResourceID: resource.id,
		Type:       resource.typ,
		State:      resource.state,
		Message:    resource.message,
	}, resource.err
}

func (resource *fakeResource) Apply(ctx context.Context) (ApplyResult, error) {
	resource.applyCalls++
	return ApplyResult{
		ResourceID: resource.id,
		Type:       resource.typ,
		Action:     string(ActionApply),
		Changed:    true,
	}, resource.applyErr
}
