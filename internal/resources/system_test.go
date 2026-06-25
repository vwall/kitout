package resources

import (
	"context"
	"testing"

	"github.com/vwall/kitout/internal/engine"
)

func TestXcodeCommandLineToolsStatusSatisfiedWhenPathExists(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout("xcode-select", []string{"-p"}, "/Library/Developer/CommandLineTools\n")},
	}}
	resource := NewXcodeCommandLineToolsRequirement(runner)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), systemType, engine.StateSatisfied, "Command Line Tools are installed")
	expectCalls(t, runner.calls, []commandCall{{name: "xcode-select", args: []string{"-p"}}})
}

func TestXcodeCommandLineToolsStatusMissingWhenSelectFails(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{err: commandError("xcode-select", []string{"-p"}, 2)},
	}}
	resource := NewXcodeCommandLineToolsRequirement(runner)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), systemType, engine.StateMissing, "Command Line Tools are missing")
}

func TestXcodeCommandLineToolsApplyStartsInstaller(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{err: commandError("xcode-select", []string{"-p"}, 2)},
		{result: commandResult("xcode-select", []string{"--install"}, 0)},
	}}
	resource := NewXcodeCommandLineToolsRequirement(runner)

	result, err := resource.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), systemType, "install", true, "started Command Line Tools installer")
	expectCalls(t, runner.calls, []commandCall{
		{name: "xcode-select", args: []string{"-p"}},
		{name: "xcode-select", args: []string{"--install"}},
	})
}

func TestXcodeCommandLineToolsDryRunDoesNotStartInstaller(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{err: commandError("xcode-select", []string{"-p"}, 2)},
	}}
	resource := NewXcodeCommandLineToolsRequirement(runner)

	plan := engine.NewPlanner().Build(context.Background(), []engine.Resource{resource})

	if plan.Items[0].Action != engine.ActionApply {
		t.Fatalf("Action = %q, want %q", plan.Items[0].Action, engine.ActionApply)
	}
	expectCalls(t, runner.calls, []commandCall{{name: "xcode-select", args: []string{"-p"}}})
}

func TestRosettaStatusSatisfiedWhenInstalledOnAppleSilicon(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout("uname", []string{"-m"}, "arm64\n")},
		{result: commandResult("pkgutil", []string{"--pkg-info", "com.apple.pkg.RosettaUpdateAuto"}, 0)},
	}}
	resource := NewRosettaRequirement(runner)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), systemType, engine.StateSatisfied, "Rosetta is installed")
	expectCalls(t, runner.calls, []commandCall{
		{name: "uname", args: []string{"-m"}},
		{name: "pkgutil", args: []string{"--pkg-info", "com.apple.pkg.RosettaUpdateAuto"}},
	})
}

func TestRosettaStatusSatisfiedWhenNotAppleSilicon(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout("uname", []string{"-m"}, "x86_64\n")},
	}}
	resource := NewRosettaRequirement(runner)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), systemType, engine.StateSatisfied, "Rosetta is not required on this architecture")
	expectCalls(t, runner.calls, []commandCall{{name: "uname", args: []string{"-m"}}})
}

func TestRosettaStatusMissingWhenReceiptMissing(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout("uname", []string{"-m"}, "arm64\n")},
		{err: commandError("pkgutil", []string{"--pkg-info", "com.apple.pkg.RosettaUpdateAuto"}, 1)},
	}}
	resource := NewRosettaRequirement(runner)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), systemType, engine.StateMissing, "Rosetta is missing")
}

func TestRosettaApplyInstallsWhenMissing(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout("uname", []string{"-m"}, "arm64\n")},
		{err: commandError("pkgutil", []string{"--pkg-info", "com.apple.pkg.RosettaUpdateAuto"}, 1)},
		{result: commandResult("softwareupdate", []string{"--install-rosetta", "--agree-to-license"}, 0)},
	}}
	resource := NewRosettaRequirement(runner)

	result, err := resource.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), systemType, "install", true, "installed Rosetta")
	expectCalls(t, runner.calls, []commandCall{
		{name: "uname", args: []string{"-m"}},
		{name: "pkgutil", args: []string{"--pkg-info", "com.apple.pkg.RosettaUpdateAuto"}},
		{name: "softwareupdate", args: []string{"--install-rosetta", "--agree-to-license"}},
	})
}

func TestRosettaApplyReportsInstallFailure(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout("uname", []string{"-m"}, "arm64\n")},
		{err: commandError("pkgutil", []string{"--pkg-info", "com.apple.pkg.RosettaUpdateAuto"}, 1)},
		{err: commandError("softwareupdate", []string{"--install-rosetta", "--agree-to-license"}, 1)},
	}}
	resource := NewRosettaRequirement(runner)

	result, err := resource.Apply(context.Background())
	if err == nil {
		t.Fatal("Apply returned nil error, want Rosetta install failure")
	}

	expectApply(t, result, resource.ID(), systemType, "install", false, "could not install Rosetta")
}
