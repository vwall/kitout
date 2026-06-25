package resources

import (
	"context"
	"testing"

	"github.com/vwall/kitout/internal/engine"
)

func TestFileVaultStatusSatisfiedWhenEnabled(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout("fdesetup", []string{"status"}, "FileVault is On.\n")},
	}}
	resource := NewFileVaultRequirement(runner)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), securityType, engine.StateSatisfied, "FileVault is enabled")
	expectCalls(t, runner.calls, []commandCall{{name: "fdesetup", args: []string{"status"}}})
}

func TestFileVaultBlocksLaterApplyWork(t *testing.T) {
	resource := NewFileVaultRequirement(&fakeRunner{})

	if !resource.BlocksApply() {
		t.Fatal("BlocksApply() = false, want true")
	}
}

func TestFileVaultStatusMissingWhenDisabled(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout("fdesetup", []string{"status"}, "FileVault is Off.\n")},
	}}
	resource := NewFileVaultRequirement(runner)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), securityType, engine.StateMissing, "FileVault must be enabled")
}

func TestFileVaultApplyIsManualWhenMissing(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout("fdesetup", []string{"status"}, "FileVault is Off.\n")},
		{result: commandResult("open", []string{"x-apple.systempreferences:com.apple.preference.security"}, 0)},
	}}
	resource := NewFileVaultRequirement(runner)

	result, err := resource.Apply(context.Background())
	if err == nil {
		t.Fatal("Apply returned nil error, want manual enablement error")
	}

	expectApply(t, result, resource.ID(), securityType, "manual", false, "enable FileVault manually in System Settings, then rerun Kitout")
	expectCalls(t, runner.calls, []commandCall{
		{name: "fdesetup", args: []string{"status"}},
		{name: "open", args: []string{"x-apple.systempreferences:com.apple.preference.security"}},
	})
}

func TestFileVaultDryRunDoesNotOpenSettings(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout("fdesetup", []string{"status"}, "FileVault is Off.\n")},
	}}
	resource := NewFileVaultRequirement(runner)

	plan := engine.NewPlanner().Build(context.Background(), []engine.Resource{resource})

	if plan.Items[0].Action != engine.ActionApply {
		t.Fatalf("Action = %q, want %q", plan.Items[0].Action, engine.ActionApply)
	}
	expectCalls(t, runner.calls, []commandCall{{name: "fdesetup", args: []string{"status"}}})
}

func TestFirewallStatusSatisfiedWhenStateMatches(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout(socketfilterfwPath, []string{"--getglobalstate"}, "Firewall is enabled. (State = 1)\n")},
	}}
	resource := NewFirewall(true, runner)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), securityType, engine.StateSatisfied, "firewall is enabled")
	expectCalls(t, runner.calls, []commandCall{{name: socketfilterfwPath, args: []string{"--getglobalstate"}}})
}

func TestFirewallStatusChangedWhenStateDiffers(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout(socketfilterfwPath, []string{"--getglobalstate"}, "Firewall is disabled. (State = 0)\n")},
	}}
	resource := NewFirewall(true, runner)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), securityType, engine.StateChanged, "firewall state differs")
}

func TestFirewallApplyUpdatesState(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout(socketfilterfwPath, []string{"--getglobalstate"}, "Firewall is disabled. (State = 0)\n")},
		{result: commandResult("sudo", []string{socketfilterfwPath, "--setglobalstate", "on"}, 0)},
	}}
	resource := NewFirewall(true, runner)

	result, err := resource.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), securityType, "set", true, "updated firewall to enabled")
	expectCalls(t, runner.calls, []commandCall{
		{name: socketfilterfwPath, args: []string{"--getglobalstate"}},
		{name: "sudo", args: []string{socketfilterfwPath, "--setglobalstate", "on"}},
	})
}

func TestFirewallApplyReportsFailure(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout(socketfilterfwPath, []string{"--getglobalstate"}, "Firewall is disabled. (State = 0)\n")},
		{err: commandError("sudo", []string{socketfilterfwPath, "--setglobalstate", "on"}, 1)},
	}}
	resource := NewFirewall(true, runner)

	result, err := resource.Apply(context.Background())
	if err == nil {
		t.Fatal("Apply returned nil error, want firewall update failure")
	}

	expectApply(t, result, resource.ID(), securityType, "set", false, "could not update firewall")
}

func TestFirewallStealthModeApplyUpdatesState(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout(socketfilterfwPath, []string{"--getstealthmode"}, "Stealth mode disabled\n")},
		{result: commandResult("sudo", []string{socketfilterfwPath, "--setstealthmode", "on"}, 0)},
	}}
	resource := NewFirewallStealthMode(true, runner)

	result, err := resource.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), securityType, "set", true, "updated firewall stealth mode to enabled")
	expectCalls(t, runner.calls, []commandCall{
		{name: socketfilterfwPath, args: []string{"--getstealthmode"}},
		{name: "sudo", args: []string{socketfilterfwPath, "--setstealthmode", "on"}},
	})
}

func TestFirewallStealthModeDryRunDoesNotUpdateState(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout(socketfilterfwPath, []string{"--getstealthmode"}, "Stealth mode disabled\n")},
	}}
	resource := NewFirewallStealthMode(true, runner)

	plan := engine.NewPlanner().Build(context.Background(), []engine.Resource{resource})

	if plan.Items[0].Action != engine.ActionApply {
		t.Fatalf("Action = %q, want %q", plan.Items[0].Action, engine.ActionApply)
	}
	expectCalls(t, runner.calls, []commandCall{{name: socketfilterfwPath, args: []string{"--getstealthmode"}}})
}
