package engine

import (
	"context"
	"errors"
	"testing"
)

func TestExecutorAppliesOnlyPlannedApplyActions(t *testing.T) {
	applyResource := &fakeResource{id: "directory:/tmp/code", typ: "directory", state: StateMissing}
	noopResource := &fakeResource{id: "brew:git", typ: "brew", state: StateSatisfied}
	resources := []Resource{applyResource, noopResource}
	plan := Plan{
		Items: []PlanItem{
			{ResourceID: "directory:/tmp/code", Type: "directory", State: StateMissing, Action: ActionApply, Message: "directory is missing"},
			{ResourceID: "brew:git", Type: "brew", State: StateSatisfied, Action: ActionNoop, Message: "formula is installed"},
		},
	}

	report := NewExecutor().Apply(context.Background(), resources, plan)

	if applyResource.applyCalls != 1 {
		t.Fatalf("applyResource.applyCalls = %d, want 1", applyResource.applyCalls)
	}
	if noopResource.applyCalls != 0 {
		t.Fatalf("noopResource.applyCalls = %d, want 0", noopResource.applyCalls)
	}
	if len(report.Items) != 2 {
		t.Fatalf("len(report.Items) = %d, want 2", len(report.Items))
	}
	if report.Items[0].ResourceID != "directory:/tmp/code" || report.Items[0].Action != "apply" {
		t.Fatalf("first apply item = %+v, want directory apply", report.Items[0])
	}
	if report.Items[1].ResourceID != "brew:git" || report.Items[1].Action != "noop" {
		t.Fatalf("second apply item = %+v, want brew noop", report.Items[1])
	}
	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false")
	}
}

func TestExecutorReportsApplyFailures(t *testing.T) {
	resource := &fakeResource{
		id:       "shell:setup",
		typ:      "shell",
		state:    StateMissing,
		applyErr: errors.New("command failed"),
	}
	plan := Plan{
		Items: []PlanItem{
			{ResourceID: "shell:setup", Type: "shell", State: StateMissing, Action: ActionApply},
		},
	}

	report := NewExecutor().Apply(context.Background(), []Resource{resource}, plan)

	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want true")
	}
	if report.Summary.Failed != 1 {
		t.Fatalf("Failed = %d, want 1", report.Summary.Failed)
	}
	if report.Items[0].Error != "command failed" {
		t.Fatalf("Error = %q, want command failed", report.Items[0].Error)
	}
}

func TestExecutorReportsProgressBeforeApply(t *testing.T) {
	resource := &fakeResource{id: "brew:go", typ: "brew", state: StateChanged}
	observer := &recordingApplyObserver{
		applyCallsFor: func() int {
			return resource.applyCalls
		},
	}
	plan := Plan{
		Items: []PlanItem{
			{ResourceID: "brew:go", Type: "brew", State: StateChanged, Action: ActionApply},
		},
	}

	NewExecutor().ApplyWithObserver(context.Background(), []Resource{resource}, plan, observer)

	if len(observer.items) != 1 {
		t.Fatalf("len(observer.items) = %d, want 1", len(observer.items))
	}
	if observer.items[0].ResourceID != "brew:go" {
		t.Fatalf("observer item = %+v, want brew:go", observer.items[0])
	}
	if observer.applyCalls[0] != 0 {
		t.Fatalf("observer saw applyCalls = %d, want 0 before apply", observer.applyCalls[0])
	}
}

type recordingApplyObserver struct {
	items         []PlanItem
	applyCalls    []int
	applyCallsFor func() int
}

func (observer *recordingApplyObserver) BeforeApply(item PlanItem) {
	observer.items = append(observer.items, item)
	if observer.applyCallsFor != nil {
		observer.applyCalls = append(observer.applyCalls, observer.applyCallsFor())
	}
}
