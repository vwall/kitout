package resources

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/vwall/kitout/internal/engine"
)

func TestShellStatusSatisfiedWhenMissingCommandExists(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{{}}}
	resource := NewShellCommand("Enable Corepack", "corepack enable", "missing-command:pnpm", runner)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), shellType, engine.StateSatisfied, "condition already satisfied")
	expectCalls(t, runner.calls, []commandCall{{name: "sh", args: []string{"-c", "command -v \"$1\"", "kitout", "pnpm"}}})
}

func TestShellStatusMissingWhenCommandShouldRun(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{{err: commandError("sh", []string{"-c", "command -v \"$1\"", "kitout", "pnpm"}, 1)}}}
	resource := NewShellCommand("Enable Corepack", "corepack enable", "missing-command:pnpm", runner)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), shellType, engine.StateMissing, "command should run")
}

func TestShellStatusMissingForAlwaysCondition(t *testing.T) {
	resource := NewShellCommand("Run setup", "setup-tool", "always", &fakeRunner{})

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), shellType, engine.StateMissing, "command should run")
}

func TestShellStatusPathConditions(t *testing.T) {
	existingPath := t.TempDir()
	missingPath := filepath.Join(t.TempDir(), "missing")

	tests := []struct {
		name      string
		when      string
		wantState engine.ResourceState
	}{
		{name: "exists true", when: "exists:" + existingPath, wantState: engine.StateMissing},
		{name: "exists false", when: "exists:" + missingPath, wantState: engine.StateSatisfied},
		{name: "missing true", when: "missing:" + missingPath, wantState: engine.StateMissing},
		{name: "missing false", when: "missing:" + existingPath, wantState: engine.StateSatisfied},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := NewShellCommand("Run setup", "setup-tool", tt.when, &fakeRunner{})

			result, err := resource.Status(context.Background())
			if err != nil {
				t.Fatalf("Status returned error: %v", err)
			}
			if result.State != tt.wantState {
				t.Fatalf("State = %q, want %q", result.State, tt.wantState)
			}
		})
	}
}

func TestShellStatusFailsForUnsupportedCondition(t *testing.T) {
	resource := NewShellCommand("Run setup", "setup-tool", "never", &fakeRunner{})

	result, err := resource.Status(context.Background())
	if !containsError(err, "unsupported shell condition") {
		t.Fatalf("Status error = %v, want unsupported condition", err)
	}

	expectStatus(t, result, resource.ID(), shellType, engine.StateFailed, "unsupported shell condition \"never\"")
}

func TestShellApplyRunsConfiguredCommand(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{{}}}
	resource := NewShellCommand("Enable Corepack", "corepack enable", "always", runner)

	result, err := resource.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), shellType, "run", true, "command completed")
	expectCalls(t, runner.calls, []commandCall{{name: "sh", args: []string{"-c", "corepack enable"}}})
}

func TestShellApplyIsIdempotentWhenConditionIsSatisfied(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{{}}}
	resource := NewShellCommand("Enable Corepack", "corepack enable", "missing-command:pnpm", runner)

	result, err := resource.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), shellType, "noop", false, "command does not need to run")
	expectCalls(t, runner.calls, []commandCall{{name: "sh", args: []string{"-c", "command -v \"$1\"", "kitout", "pnpm"}}})
}

func TestShellApplyReportsCommandFailure(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{{err: commandError("sh", []string{"-c", "corepack enable"}, 17)}}}
	resource := NewShellCommand("Enable Corepack", "corepack enable", "always", runner)

	result, err := resource.Apply(context.Background())
	if err == nil {
		t.Fatal("Apply returned nil error, want command failure")
	}

	expectApply(t, result, resource.ID(), shellType, "run", false, "command failed")
}

func TestShellDryRunPlanDoesNotRunCommand(t *testing.T) {
	runner := &fakeRunner{}
	resource := NewShellCommand("Enable Corepack", "corepack enable", "always", runner)

	plan := engine.NewPlanner().Build(context.Background(), []engine.Resource{resource})

	if plan.Items[0].Action != engine.ActionApply {
		t.Fatalf("Action = %q, want %q", plan.Items[0].Action, engine.ActionApply)
	}
	expectCalls(t, runner.calls, nil)
}
