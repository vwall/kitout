package resources

import (
	"context"
	"testing"

	"github.com/vwall/kitout/internal/engine"
)

func TestMacOSDefaultStatusSatisfiedForSupportedTypes(t *testing.T) {
	tests := []struct {
		name      string
		valueType string
		value     any
		stdout    string
	}{
		{name: "bool", valueType: "bool", value: true, stdout: "1\n"},
		{name: "int", valueType: "int", value: 42, stdout: "42\n"},
		{name: "float", valueType: "float", value: 1.5, stdout: "1.500000\n"},
		{name: "string", valueType: "string", value: "compact", stdout: "compact\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &fakeRunner{responses: []fakeResponse{
				{result: resultWithStdout("defaults", []string{"read", "NSGlobalDomain", "KitoutTestKey"}, tt.stdout)},
			}}
			resource := NewMacOSDefault("NSGlobalDomain", "KitoutTestKey", tt.valueType, tt.value, runner)

			result, err := resource.Status(context.Background())
			if err != nil {
				t.Fatalf("Status returned error: %v", err)
			}

			expectStatus(t, result, resource.ID(), macOSDefaultType, engine.StateSatisfied, "default is set")
			expectCalls(t, runner.calls, []commandCall{
				{name: "defaults", args: []string{"read", "NSGlobalDomain", "KitoutTestKey"}},
			})
		})
	}
}

func TestMacOSDefaultStatusChangedWhenValueDiffers(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout("defaults", []string{"read", "NSGlobalDomain", "AppleShowAllExtensions"}, "0\n")},
	}}
	resource := NewMacOSDefault("NSGlobalDomain", "AppleShowAllExtensions", "bool", true, runner)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), macOSDefaultType, engine.StateChanged, "default value differs")
}

func TestMacOSDefaultStatusMissingWhenDefaultsReadFailsWithExitOne(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{err: commandError("defaults", []string{"read", "NSGlobalDomain", "MissingKey"}, 1)},
	}}
	resource := NewMacOSDefault("NSGlobalDomain", "MissingKey", "string", "value", runner)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), macOSDefaultType, engine.StateMissing, "default is missing")
}

func TestMacOSDefaultStatusReportsUnexpectedReadFailure(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{err: commandError("defaults", []string{"read", "NSGlobalDomain", "KitoutTestKey"}, 2)},
	}}
	resource := NewMacOSDefault("NSGlobalDomain", "KitoutTestKey", "int", 42, runner)

	result, err := resource.Status(context.Background())
	if err == nil {
		t.Fatal("Status returned nil error, want read failure")
	}

	expectStatus(t, result, resource.ID(), macOSDefaultType, engine.StateFailed, "could not inspect default")
}

func TestMacOSDefaultApplyWritesMissingDefaultsForSupportedTypes(t *testing.T) {
	tests := []struct {
		name      string
		valueType string
		value     any
		flag      string
		arg       string
	}{
		{name: "bool", valueType: "bool", value: true, flag: "-bool", arg: "true"},
		{name: "int", valueType: "int", value: 42, flag: "-int", arg: "42"},
		{name: "float", valueType: "float", value: 1.5, flag: "-float", arg: "1.5"},
		{name: "string", valueType: "string", value: "compact", flag: "-string", arg: "compact"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &fakeRunner{responses: []fakeResponse{
				{err: commandError("defaults", []string{"read", "NSGlobalDomain", "KitoutTestKey"}, 1)},
				{result: commandResult("defaults", []string{"write", "NSGlobalDomain", "KitoutTestKey", tt.flag, tt.arg}, 0)},
			}}
			resource := NewMacOSDefault("NSGlobalDomain", "KitoutTestKey", tt.valueType, tt.value, runner)

			result, err := resource.Apply(context.Background())
			if err != nil {
				t.Fatalf("Apply returned error: %v", err)
			}

			expectApply(t, result, resource.ID(), macOSDefaultType, "write", true, "wrote default")
			expectCalls(t, runner.calls, []commandCall{
				{name: "defaults", args: []string{"read", "NSGlobalDomain", "KitoutTestKey"}},
				{name: "defaults", args: []string{"write", "NSGlobalDomain", "KitoutTestKey", tt.flag, tt.arg}},
			})
		})
	}
}

func TestMacOSDefaultApplyIsIdempotentWhenDefaultIsSet(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout("defaults", []string{"read", "NSGlobalDomain", "AppleShowAllExtensions"}, "1\n")},
	}}
	resource := NewMacOSDefault("NSGlobalDomain", "AppleShowAllExtensions", "bool", true, runner)

	result, err := resource.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), macOSDefaultType, "noop", false, "default already set")
	expectCalls(t, runner.calls, []commandCall{
		{name: "defaults", args: []string{"read", "NSGlobalDomain", "AppleShowAllExtensions"}},
	})
}

func TestMacOSDefaultApplyReportsWriteFailure(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{err: commandError("defaults", []string{"read", "NSGlobalDomain", "AppleShowAllExtensions"}, 1)},
		{err: commandError("defaults", []string{"write", "NSGlobalDomain", "AppleShowAllExtensions", "-bool", "true"}, 2)},
	}}
	resource := NewMacOSDefault("NSGlobalDomain", "AppleShowAllExtensions", "bool", true, runner)

	result, err := resource.Apply(context.Background())
	if err == nil {
		t.Fatal("Apply returned nil error, want write failure")
	}

	expectApply(t, result, resource.ID(), macOSDefaultType, "write", false, "could not write default")
}

func TestMacOSDefaultDryRunPlanDoesNotWriteDefault(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{err: commandError("defaults", []string{"read", "NSGlobalDomain", "AppleShowAllExtensions"}, 1)},
	}}
	resource := NewMacOSDefault("NSGlobalDomain", "AppleShowAllExtensions", "bool", true, runner)

	plan := engine.NewPlanner().Build(context.Background(), []engine.Resource{resource})

	if plan.Items[0].Action != engine.ActionApply {
		t.Fatalf("Action = %q, want %q", plan.Items[0].Action, engine.ActionApply)
	}
	expectCalls(t, runner.calls, []commandCall{
		{name: "defaults", args: []string{"read", "NSGlobalDomain", "AppleShowAllExtensions"}},
	})
}
