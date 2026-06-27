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

func TestExecutorSkipsRemainingApplyActionsAfterBlockingFailure(t *testing.T) {
	blocker := &fakeResource{
		id:          "security:filevault",
		typ:         "security",
		state:       StateMissing,
		applyErr:    errors.New("enable FileVault manually"),
		blocksApply: true,
	}
	directory := &fakeResource{id: "directory:/tmp/code", typ: "directory", state: StateMissing}
	noop := &fakeResource{id: "brew:git", typ: "brew", state: StateSatisfied}
	plan := Plan{
		Items: []PlanItem{
			{ResourceID: "security:filevault", Type: "security", State: StateMissing, Action: ActionApply},
			{ResourceID: "directory:/tmp/code", Type: "directory", State: StateMissing, Action: ActionApply},
			{ResourceID: "brew:git", Type: "brew", State: StateSatisfied, Action: ActionNoop, Message: "formula is installed"},
		},
	}

	report := NewExecutor().Apply(context.Background(), []Resource{blocker, directory, noop}, plan)

	if blocker.applyCalls != 1 {
		t.Fatalf("blocker.applyCalls = %d, want 1", blocker.applyCalls)
	}
	if directory.applyCalls != 0 {
		t.Fatalf("directory.applyCalls = %d, want 0", directory.applyCalls)
	}
	if len(report.Items) != 3 {
		t.Fatalf("len(report.Items) = %d, want 3", len(report.Items))
	}
	if report.Items[0].Error != "enable FileVault manually" {
		t.Fatalf("blocker error = %q, want manual FileVault error", report.Items[0].Error)
	}
	if report.Items[1].Action != "skip" || report.Items[1].Message != "blocked by security:filevault" {
		t.Fatalf("blocked item = %+v, want skip blocked by security:filevault", report.Items[1])
	}
	if report.Items[2].Action != "noop" {
		t.Fatalf("noop item action = %q, want noop", report.Items[2].Action)
	}
	if report.Summary.Failed != 1 || report.Summary.Skipped != 1 {
		t.Fatalf("summary = %+v, want 1 failed and 1 skipped", report.Summary)
	}
}

func TestExecutorSkipsRemainingApplyActionsAfterBlockingPlanFailure(t *testing.T) {
	blocker := &fakeResource{
		id:          "security:filevault",
		typ:         "security",
		state:       StateFailed,
		blocksApply: true,
	}
	directory := &fakeResource{id: "directory:/tmp/code", typ: "directory", state: StateMissing}
	plan := Plan{
		Items: []PlanItem{
			{ResourceID: "security:filevault", Type: "security", State: StateFailed, Action: ActionFail, Message: "could not inspect FileVault", Error: "fdesetup failed"},
			{ResourceID: "directory:/tmp/code", Type: "directory", State: StateMissing, Action: ActionApply},
		},
	}

	report := NewExecutor().Apply(context.Background(), []Resource{blocker, directory}, plan)

	if blocker.applyCalls != 0 {
		t.Fatalf("blocker.applyCalls = %d, want 0", blocker.applyCalls)
	}
	if directory.applyCalls != 0 {
		t.Fatalf("directory.applyCalls = %d, want 0", directory.applyCalls)
	}
	if len(report.Items) != 2 {
		t.Fatalf("len(report.Items) = %d, want 2", len(report.Items))
	}
	if report.Items[0].Action != "fail" {
		t.Fatalf("blocker action = %q, want fail", report.Items[0].Action)
	}
	if report.Items[1].Action != "skip" || report.Items[1].Message != "blocked by security:filevault" {
		t.Fatalf("blocked item = %+v, want skip blocked by security:filevault", report.Items[1])
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

func TestExecutorRejectsDuplicateResourceIDsBeforeApply(t *testing.T) {
	first := &fakeResource{id: "copy:/tmp/kitout-target", typ: "copy", state: StateMissing}
	second := &fakeResource{id: "copy:/tmp/kitout-target", typ: "copy", state: StateMissing}
	plan := Plan{
		Items: []PlanItem{
			{ResourceID: "copy:/tmp/kitout-target", Type: "copy", State: StateMissing, Action: ActionApply},
		},
	}

	report := NewExecutor().Apply(context.Background(), []Resource{first, second}, plan)

	if first.applyCalls != 0 || second.applyCalls != 0 {
		t.Fatalf("applyCalls = %d, %d; want no apply calls", first.applyCalls, second.applyCalls)
	}
	if len(report.Items) != 2 {
		t.Fatalf("len(report.Items) = %d, want 2", len(report.Items))
	}
	for _, item := range report.Items {
		want := `duplicate resource ID "copy:/tmp/kitout-target"; resource IDs must be unique`
		if item.ResourceID != "copy:/tmp/kitout-target" || item.Type != "copy" || item.Action != "fail" {
			t.Fatalf("duplicate item = %+v, want copy failure", item)
		}
		if item.Error != want || item.Message != want {
			t.Fatalf("duplicate item = %+v, want error %q", item, want)
		}
	}
	if report.Summary.Failed != 2 {
		t.Fatalf("summary = %+v, want 2 failed", report.Summary)
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
