package engine

import (
	"context"
	"errors"
	"slices"
	"testing"
)

func TestPlannerBuildsPlanFromStatusResults(t *testing.T) {
	resources := []Resource{
		&fakeResource{id: "directory:/Users/example/code", typ: "directory", state: StateSatisfied, message: "exists"},
		&fakeResource{id: "brew:git", typ: "brew", state: StateMissing, message: "not installed"},
		&fakeResource{id: "symlink:/Users/example/.zshrc", typ: "symlink", state: StateChanged, message: "points elsewhere"},
		&fakeResource{id: "shell:corepack", typ: "shell", state: StateSkipped, message: "condition not met"},
	}

	plan := NewPlanner().Build(context.Background(), resources)

	if got, want := len(plan.Items), 4; got != want {
		t.Fatalf("len(plan.Items) = %d, want %d", got, want)
	}
	assertPlanItem(t, plan.Items[0], "directory:/Users/example/code", "directory", StateSatisfied, ActionNoop)
	assertPlanItem(t, plan.Items[1], "brew:git", "brew", StateMissing, ActionApply)
	assertPlanItem(t, plan.Items[2], "symlink:/Users/example/.zshrc", "symlink", StateChanged, ActionApply)
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
		&fakeResource{id: "repo:/Users/example/code/example-project", typ: "repo", err: errors.New("git not found")},
	}

	plan := NewPlanner().Build(context.Background(), resources)

	if got, want := len(plan.Items), 2; got != want {
		t.Fatalf("len(plan.Items) = %d, want %d", got, want)
	}
	assertPlanItem(t, plan.Items[1], "repo:/Users/example/code/example-project", "repo", StateFailed, ActionFail)
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

func TestPlannerCarriesAdvisoriesWithoutApplying(t *testing.T) {
	resources := []Resource{
		&fakeResource{
			id:      "brew:git",
			typ:     "brew",
			state:   StateSatisfied,
			message: "formula is installed",
			advisories: []Advisory{{
				Code:     "homebrew_formula_outdated",
				Severity: AdvisoryNotice,
				Message:  "formula update available for git",
			}},
		},
	}

	plan := NewPlanner().Build(context.Background(), resources)

	assertPlanItem(t, plan.Items[0], "brew:git", "brew", StateSatisfied, ActionNoop)
	if plan.Summary.Advisories != 1 {
		t.Fatalf("Advisories = %d, want 1", plan.Summary.Advisories)
	}
	if plan.HasChanges() {
		t.Fatalf("HasChanges() = true, want false")
	}
}

func TestPlannerNeverAppliesResources(t *testing.T) {
	resource := &fakeResource{
		id:    "directory:/Users/example/code",
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

func TestPlannerNotifiesObserverBeforeEachStatusCheck(t *testing.T) {
	resources := []Resource{
		&fakeResource{id: "directory:/Users/example/code", typ: "directory", state: StateSatisfied},
		&fakeResource{id: "brew:git", typ: "brew", state: StateMissing},
	}
	observer := &fakePlanObserver{}

	plan := NewPlanner().BuildWithObserver(context.Background(), resources, observer)

	if got, want := observer.ids, []string{"directory:/Users/example/code", "brew:git"}; !slices.Equal(got, want) {
		t.Fatalf("observer ids = %#v, want %#v", got, want)
	}
	if got, want := len(plan.Items), 2; got != want {
		t.Fatalf("len(plan.Items) = %d, want %d", got, want)
	}
}

func TestPlannerDoesNotDispatchResourcesAfterCancellation(t *testing.T) {
	first := &fakeResource{id: "directory:/tmp/first", typ: "directory", state: StateMissing}
	second := &fakeResource{id: "directory:/tmp/second", typ: "directory", state: StateMissing}
	observer := &fakePlanObserver{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	plan := NewPlanner().BuildWithObserver(ctx, []Resource{first, second}, observer)

	if first.statusCalls != 0 || second.statusCalls != 0 {
		t.Fatalf("status calls = %d, %d; want no dispatch after cancellation", first.statusCalls, second.statusCalls)
	}
	if len(observer.ids) != 0 {
		t.Fatalf("observer ids = %#v, want no notifications after cancellation", observer.ids)
	}
	assertCanceledPlan(t, plan, first.ID(), second.ID())
}

func TestPlannerPreservesCancellationWithoutResources(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	plan := NewPlanner().Build(ctx, nil)

	if len(plan.Items) != 0 {
		t.Fatalf("len(plan.Items) = %d, want 0", len(plan.Items))
	}
	if plan.ExecutionError != context.Canceled.Error() || !plan.HasFailures() {
		t.Fatalf("plan = %+v, want cancellation failure", plan)
	}
}

func TestPlannerStopsWhenStatusCancelsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	first := &fakeResource{
		id:         "directory:/tmp/first",
		typ:        "directory",
		state:      StateSatisfied,
		statusHook: cancel,
	}
	second := &fakeResource{id: "directory:/tmp/second", typ: "directory", state: StateMissing}

	plan := NewPlanner().Build(ctx, []Resource{first, second})

	if first.statusCalls != 1 || second.statusCalls != 0 {
		t.Fatalf("status calls = %d, %d; want cancellation to stop after first", first.statusCalls, second.statusCalls)
	}
	assertCanceledPlan(t, plan, first.ID(), second.ID())
}

func TestPlannerStopsWhenStatusReturnsCancellation(t *testing.T) {
	first := &fakeResource{id: "directory:/tmp/first", typ: "directory", err: context.Canceled}
	second := &fakeResource{id: "directory:/tmp/second", typ: "directory", state: StateMissing}

	plan := NewPlanner().Build(context.Background(), []Resource{first, second})

	if first.statusCalls != 1 || second.statusCalls != 0 {
		t.Fatalf("status calls = %d, %d; want returned cancellation to stop after first", first.statusCalls, second.statusCalls)
	}
	assertCanceledPlan(t, plan, first.ID(), second.ID())
}

func TestPlannerRechecksCancellationAfterObserver(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	first := &fakeResource{id: "directory:/tmp/first", typ: "directory", state: StateMissing}
	second := &fakeResource{id: "directory:/tmp/second", typ: "directory", state: StateMissing}
	observer := &fakePlanObserver{before: func(Resource) { cancel() }}

	plan := NewPlanner().BuildWithObserver(ctx, []Resource{first, second}, observer)

	if first.statusCalls != 0 || second.statusCalls != 0 {
		t.Fatalf("status calls = %d, %d; want no dispatch after observer cancellation", first.statusCalls, second.statusCalls)
	}
	if got, want := observer.ids, []string{first.ID()}; !slices.Equal(got, want) {
		t.Fatalf("observer ids = %#v, want %#v", got, want)
	}
	assertCanceledPlan(t, plan, first.ID(), second.ID())
}

func assertCanceledPlan(t *testing.T, plan Plan, resourceIDs ...string) {
	t.Helper()

	if len(plan.Items) != len(resourceIDs) {
		t.Fatalf("len(plan.Items) = %d, want %d", len(plan.Items), len(resourceIDs))
	}
	for i, resourceID := range resourceIDs {
		item := plan.Items[i]
		if item.ResourceID != resourceID || item.State != StateFailed || item.Action != ActionFail {
			t.Fatalf("item[%d] = %+v, want canceled failure for %s", i, item, resourceID)
		}
		if item.Message != "status check canceled" || item.Error != context.Canceled.Error() {
			t.Fatalf("item[%d] = %+v, want context cancellation details", i, item)
		}
	}
	if plan.Summary.Failed != len(resourceIDs) || !plan.HasFailures() {
		t.Fatalf("summary = %+v, want %d canceled failures", plan.Summary, len(resourceIDs))
	}
	if plan.ExecutionError != context.Canceled.Error() {
		t.Fatalf("ExecutionError = %q, want context canceled", plan.ExecutionError)
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

func TestPlannerUsesResourceIdentityInsteadOfResultIdentity(t *testing.T) {
	resource := &fakeResource{
		id:         "directory:/tmp/code",
		typ:        "directory",
		state:      StateMissing,
		statusID:   "brew:git",
		statusType: "brew",
	}

	plan := NewPlanner().Build(context.Background(), []Resource{resource})

	assertPlanItem(t, plan.Items[0], "directory:/tmp/code", "directory", StateMissing, ActionApply)
}

func TestPlannerRejectsDuplicateResourceIDsBeforeStatus(t *testing.T) {
	first := &fakeResource{id: "copy:/tmp/kitout-target", typ: "copy", state: StateMissing}
	second := &fakeResource{id: "copy:/tmp/kitout-target", typ: "copy", state: StateSatisfied}

	plan := NewPlanner().Build(context.Background(), []Resource{first, second})

	if first.statusCalls != 0 || second.statusCalls != 0 {
		t.Fatalf("statusCalls = %d, %d; want no status calls", first.statusCalls, second.statusCalls)
	}
	if len(plan.Items) != 2 {
		t.Fatalf("len(plan.Items) = %d, want 2", len(plan.Items))
	}
	for _, item := range plan.Items {
		assertPlanItem(t, item, "copy:/tmp/kitout-target", "copy", StateFailed, ActionFail)
		want := `duplicate resource ID "copy:/tmp/kitout-target"; resource IDs must be unique`
		if item.Error != want || item.Message != want {
			t.Fatalf("duplicate item = %+v, want error %q", item, want)
		}
	}
	if plan.Summary.Failed != 2 || plan.Summary.ToApply != 0 {
		t.Fatalf("summary = %+v, want 2 failed and 0 to apply", plan.Summary)
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
	statusID    string
	statusType  string
	applyID     string
	applyType   string
	state       ResourceState
	message     string
	err         error
	applyErr    error
	applyHook   func()
	statusHook  func()
	advisories  []Advisory
	blocksApply bool
	statusCalls int
	applyCalls  int
}

type fakePlanObserver struct {
	ids    []string
	before func(Resource)
}

func (observer *fakePlanObserver) BeforeStatus(resource Resource) {
	observer.ids = append(observer.ids, resource.ID())
	if observer.before != nil {
		observer.before(resource)
	}
}

func (resource *fakeResource) ID() string {
	return resource.id
}

func (resource *fakeResource) Type() string {
	return resource.typ
}

func (resource *fakeResource) Status(ctx context.Context) (StatusResult, error) {
	resource.statusCalls++
	if resource.statusHook != nil {
		resource.statusHook()
	}
	id, typ := resource.id, resource.typ
	if resource.statusID != "" {
		id = resource.statusID
	}
	if resource.statusType != "" {
		typ = resource.statusType
	}
	return StatusResult{
		ResourceID: id,
		Type:       typ,
		State:      resource.state,
		Message:    resource.message,
		Advisories: resource.advisories,
	}, resource.err
}

func (resource *fakeResource) Apply(ctx context.Context) (ApplyResult, error) {
	resource.applyCalls++
	if resource.applyHook != nil {
		resource.applyHook()
	}
	id, typ := resource.id, resource.typ
	if resource.applyID != "" {
		id = resource.applyID
	}
	if resource.applyType != "" {
		typ = resource.applyType
	}
	return ApplyResult{
		ResourceID: id,
		Type:       typ,
		Action:     string(ActionApply),
		Changed:    true,
	}, resource.applyErr
}

func (resource *fakeResource) BlocksApply() bool {
	return resource.blocksApply
}
